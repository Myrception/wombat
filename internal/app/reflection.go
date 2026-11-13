package app

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var errInvalidReflectionResponse = errors.New("invalid response from the server reflection service")

// protoFilesFromReflectionAPI loads the schema of every service the server
// lists through the gRPC server reflection API (v1, or v1alpha for older
// servers).
//
// Each request is a network round trip, which adds up quickly on a remote
// server, so the loader asks for as little as possible: servers send a file
// together with the dependencies they have not sent on the stream yet, so
// everything a response contains is recorded first and only files that are
// still missing afterwards are requested by name.
func protoFilesFromReflectionAPI(ctx context.Context, conn *grpc.ClientConn) (*protoregistry.Files, error) {
	if conn == nil {
		return nil, errors.New("no connection to a gRPC server available")
	}

	// Cancelling the context is what fully releases the stream.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rs, services, err := openReflection(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	defer func() { _ = rs.CloseSend() }() // cancelling ctx releases the stream anyway

	l := newFileLoader()
	for _, svc := range services {
		if l.services[svc] {
			continue // defined in a file we already have
		}
		raw, err := rs.files(svc, "")
		if err != nil {
			return nil, fmt.Errorf("loading service %s: %w", svc, err)
		}
		if err := l.add(raw); err != nil {
			return nil, fmt.Errorf("loading service %s: %w", svc, err)
		}
	}

	for missing := l.missing(); len(missing) > 0; missing = l.missing() {
		for _, name := range missing {
			raw, err := rs.files("", name)
			if err != nil {
				return nil, fmt.Errorf("loading %s: %w", name, err)
			}
			if err := l.add(raw); err != nil {
				return nil, fmt.Errorf("loading %s: %w", name, err)
			}
			if l.files[name] == nil {
				// Guarantees progress: without this a misbehaving server
				// could keep us asking for the same file forever.
				return nil, fmt.Errorf("loading %s: the server did not return it", name)
			}
		}
	}

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: l.ordered})
	if err != nil {
		return nil, fmt.Errorf("building the schema: %w", err)
	}
	return files, nil
}

// fileLoader collects file descriptors received over a reflection stream.
type fileLoader struct {
	files    map[string]*descriptorpb.FileDescriptorProto
	ordered  []*descriptorpb.FileDescriptorProto // in the order received
	services map[string]bool                     // fully qualified services defined by files
}

func newFileLoader() *fileLoader {
	return &fileLoader{
		files:    map[string]*descriptorpb.FileDescriptorProto{},
		services: map[string]bool{},
	}
}

func (l *fileLoader) add(raw [][]byte) error {
	for _, b := range raw {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(b, fd); err != nil {
			return fmt.Errorf("decoding file descriptor: %w", err)
		}
		name := fd.GetName()
		if name == "" || l.files[name] != nil {
			continue
		}
		l.files[name] = fd
		l.ordered = append(l.ordered, fd)
		for _, sd := range fd.GetService() {
			fq := sd.GetName()
			if pkg := fd.GetPackage(); pkg != "" {
				fq = pkg + "." + fq
			}
			l.services[fq] = true
		}
	}
	return nil
}

// missing returns the dependencies of the received files that have not been
// received themselves.
func (l *fileLoader) missing() []string {
	var out []string
	seen := map[string]bool{}
	for _, fd := range l.ordered {
		for _, dep := range fd.GetDependency() {
			if l.files[dep] == nil && !seen[dep] {
				seen[dep] = true
				out = append(out, dep)
			}
		}
	}
	return out
}

// reflectionStream is one ServerReflectionInfo stream of either API version.
type reflectionStream interface {
	listServices() ([]string, error)
	// files requests the file defining symbol, or the file named filename.
	files(symbol, filename string) ([][]byte, error)
	CloseSend() error
}

// openReflection opens a reflection stream and lists the services, using v1
// and falling back to v1alpha for servers that only implement the latter.
func openReflection(ctx context.Context, conn *grpc.ClientConn) (reflectionStream, []string, error) {
	v1, err := reflectionv1.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, nil, err
	}
	rs := reflectionStream(v1Stream{v1})
	services, err := rs.listServices()
	if status.Code(err) == codes.Unimplemented {
		_ = v1.CloseSend() // the server rejected it already
		alpha, aerr := reflectionv1alpha.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
		if aerr != nil {
			return nil, nil, aerr
		}
		rs = v1alphaStream{alpha}
		services, aerr = rs.listServices()
		if status.Code(aerr) == codes.Unimplemented {
			return nil, nil, fmt.Errorf("the server does not support reflection: %w", err)
		}
		err = aerr
	}
	if err != nil {
		return nil, nil, err
	}
	return rs, services, nil
}

type v1Stream struct {
	reflectionv1.ServerReflection_ServerReflectionInfoClient
}

func (s v1Stream) roundTrip(req *reflectionv1.ServerReflectionRequest) (*reflectionv1.ServerReflectionResponse, error) {
	if err := s.Send(req); err != nil {
		return nil, err
	}
	resp, err := s.Recv()
	if err != nil {
		return nil, err
	}
	if e := resp.GetErrorResponse(); e != nil {
		return nil, status.Error(codes.Code(e.GetErrorCode()), e.GetErrorMessage())
	}
	return resp, nil
}

func (s v1Stream) listServices() ([]string, error) {
	resp, err := s.roundTrip(&reflectionv1.ServerReflectionRequest{
		MessageRequest: &reflectionv1.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		return nil, err
	}
	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, errInvalidReflectionResponse
	}
	names := make([]string, 0, len(list.GetService()))
	for _, svc := range list.GetService() {
		names = append(names, svc.GetName())
	}
	return names, nil
}

func (s v1Stream) files(symbol, filename string) ([][]byte, error) {
	req := &reflectionv1.ServerReflectionRequest{}
	if symbol != "" {
		req.MessageRequest = &reflectionv1.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol}
	} else {
		req.MessageRequest = &reflectionv1.ServerReflectionRequest_FileByFilename{FileByFilename: filename}
	}
	resp, err := s.roundTrip(req)
	if err != nil {
		return nil, err
	}
	fds := resp.GetFileDescriptorResponse()
	if fds == nil {
		return nil, errInvalidReflectionResponse
	}
	return fds.GetFileDescriptorProto(), nil
}

type v1alphaStream struct {
	reflectionv1alpha.ServerReflection_ServerReflectionInfoClient
}

func (s v1alphaStream) roundTrip(req *reflectionv1alpha.ServerReflectionRequest) (*reflectionv1alpha.ServerReflectionResponse, error) {
	if err := s.Send(req); err != nil {
		return nil, err
	}
	resp, err := s.Recv()
	if err != nil {
		return nil, err
	}
	if e := resp.GetErrorResponse(); e != nil {
		return nil, status.Error(codes.Code(e.GetErrorCode()), e.GetErrorMessage())
	}
	return resp, nil
}

func (s v1alphaStream) listServices() ([]string, error) {
	resp, err := s.roundTrip(&reflectionv1alpha.ServerReflectionRequest{
		MessageRequest: &reflectionv1alpha.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		return nil, err
	}
	list := resp.GetListServicesResponse()
	if list == nil {
		return nil, errInvalidReflectionResponse
	}
	names := make([]string, 0, len(list.GetService()))
	for _, svc := range list.GetService() {
		names = append(names, svc.GetName())
	}
	return names, nil
}

func (s v1alphaStream) files(symbol, filename string) ([][]byte, error) {
	req := &reflectionv1alpha.ServerReflectionRequest{}
	if symbol != "" {
		req.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: symbol}
	} else {
		req.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_FileByFilename{FileByFilename: filename}
	}
	resp, err := s.roundTrip(req)
	if err != nil {
		return nil, err
	}
	fds := resp.GetFileDescriptorResponse()
	if fds == nil {
		return nil, errInvalidReflectionResponse
	}
	return fds.GetFileDescriptorProto(), nil
}
