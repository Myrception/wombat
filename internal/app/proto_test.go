package app

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"wombat/internal/server"
)

// reflectionCounter counts the reflection requests a server receives, by kind.
type reflectionCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

func (c *reflectionCounter) add(kind string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.counts == nil {
		c.counts = map[string]int{}
	}
	c.counts[kind]++
}

func (c *reflectionCounter) get(kind string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[kind]
}

type countingStream struct {
	grpc.ServerStream
	c *reflectionCounter
}

func (s countingStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		return err
	}
	switch r := m.(type) {
	case *reflectionv1.ServerReflectionRequest:
		s.c.add(requestKind(r.GetFileContainingSymbol(), r.GetFileByFilename()))
	case *reflectionv1alpha.ServerReflectionRequest:
		s.c.add("v1alpha:" + requestKind(r.GetFileContainingSymbol(), r.GetFileByFilename()))
	}
	return nil
}

func requestKind(symbol, filename string) string {
	switch {
	case symbol != "":
		return "symbol"
	case filename != "":
		return "filename"
	default:
		return "list"
	}
}

func (c *reflectionCounter) interceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, countingStream{ss, c})
	}
}

func dialTest(t *testing.T, addr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func loadCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func wantServices(t *testing.T, files interface {
	FindDescriptorByName(protoreflect.FullName) (protoreflect.Descriptor, error)
}, names ...string) {
	t.Helper()
	for _, n := range names {
		d, err := files.FindDescriptorByName(protoreflect.FullName(n))
		if err != nil {
			t.Errorf("service %s not loaded: %v", n, err)
			continue
		}
		if _, ok := d.(protoreflect.ServiceDescriptor); !ok {
			t.Errorf("%s is a %T, not a service", n, d)
		}
	}
}

// The reflection server already sends every file together with its
// dependencies, so loading must not re-request files it has received.
func TestReflectionNoRedundantRequests(t *testing.T) {
	var c reflectionCounter
	addr, _ := startServer(t, grpc.StreamInterceptor(c.interceptor()))

	files, err := protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, addr))
	if err != nil {
		t.Fatal(err)
	}
	wantServices(t, files, "wombat.v1.RouteGuide", "wombat.v1.Foobar", "grpc.reflection.v1.ServerReflection")

	if n := c.get("filename"); n != 0 {
		t.Errorf("requested %d files by name, want 0 (all dependencies were already sent)", n)
	}
	// RouteGuide and Foobar live in two files; each reflection service in one more.
	if n := c.get("symbol"); n > 4 {
		t.Errorf("requested %d files by symbol, want at most 4", n)
	}
	t.Logf("requests: %v", c.counts)
}

// Older servers only implement v1alpha.
func TestReflectionV1AlphaFallback(t *testing.T) {
	var c reflectionCounter
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer(grpc.StreamInterceptor(c.interceptor()))
	server.RegisterRouteGuideServer(gs, nil)
	reflectionv1alpha.RegisterServerReflectionServer(gs, reflection.NewServer(reflection.ServerOptions{Services: gs}))
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	files, err := protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, lis.Addr().String()))
	if err != nil {
		t.Fatal(err)
	}
	wantServices(t, files, "wombat.v1.RouteGuide")
	if c.get("v1alpha:list") == 0 {
		t.Error("v1alpha was not used")
	}
}

// A server without reflection must give a clear error, not hang or panic.
func TestReflectionUnavailable(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer()
	server.RegisterRouteGuideServer(gs, nil)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)

	_, err = protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, lis.Addr().String()))
	if err == nil {
		t.Fatal("expected an error")
	}
	t.Logf("error: %v", err)
}

// singleFileReflection is a minimal v1 reflection server that, unlike
// grpc-go's, returns only the requested file and never its dependencies.
// If dropFile is set it also "forgets" that file when asked for it by name.
type singleFileReflection struct {
	reflectionv1.UnimplementedServerReflectionServer
	services []string
	dropFile string
}

func (s singleFileReflection) ServerReflectionInfo(stream reflectionv1.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return nil
		}
		resp := &reflectionv1.ServerReflectionResponse{}
		file := func(fd protoreflect.FileDescriptor) {
			b, _ := proto.Marshal(protodesc.ToFileDescriptorProto(fd))
			resp.MessageResponse = &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{
				FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{FileDescriptorProto: [][]byte{b}},
			}
		}
		switch {
		case req.GetFileContainingSymbol() != "":
			d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(req.GetFileContainingSymbol()))
			if err != nil {
				resp.MessageResponse = &reflectionv1.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &reflectionv1.ErrorResponse{ErrorCode: int32(codes.NotFound), ErrorMessage: err.Error()},
				}
				break
			}
			file(d.ParentFile())
		case req.GetFileByFilename() != "":
			if req.GetFileByFilename() == s.dropFile {
				resp.MessageResponse = &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{}}
				break
			}
			fd, err := protoregistry.GlobalFiles.FindFileByPath(req.GetFileByFilename())
			if err != nil {
				resp.MessageResponse = &reflectionv1.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &reflectionv1.ErrorResponse{ErrorCode: int32(codes.NotFound), ErrorMessage: err.Error()},
				}
				break
			}
			file(fd)
		default:
			list := &reflectionv1.ListServiceResponse{}
			for _, n := range s.services {
				list.Service = append(list.Service, &reflectionv1.ServiceResponse{Name: n})
			}
			resp.MessageResponse = &reflectionv1.ServerReflectionResponse_ListServicesResponse{ListServicesResponse: list}
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

func startSingleFileReflection(t *testing.T, srv singleFileReflection) (string, *reflectionCounter) {
	t.Helper()
	c := &reflectionCounter{}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer(grpc.StreamInterceptor(c.interceptor()))
	reflectionv1.RegisterServerReflectionServer(gs, srv)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)
	return lis.Addr().String(), c
}

// Dependencies the server does not send along are fetched by name.
func TestReflectionFetchesMissingDependencies(t *testing.T) {
	addr, c := startSingleFileReflection(t, singleFileReflection{services: []string{"wombat.v1.Foobar"}})

	files, err := protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, addr))
	if err != nil {
		t.Fatal(err)
	}
	wantServices(t, files, "wombat.v1.Foobar")
	if _, err := files.FindDescriptorByName("google.protobuf.Timestamp"); err != nil {
		t.Errorf("dependency not loaded: %v", err)
	}
	// foobar.proto imports timestamp, duration, wrappers and struct.
	if n := c.get("filename"); n != 4 {
		t.Errorf("requested %d files by name, want 4", n)
	}
}

// A server that never returns a dependency must produce an error, not a loop.
func TestReflectionMissingDependencyError(t *testing.T) {
	addr, _ := startSingleFileReflection(t, singleFileReflection{
		services: []string{"wombat.v1.Foobar"},
		dropFile: "google/protobuf/struct.proto",
	})

	_, err := protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, addr))
	if err == nil || !strings.Contains(err.Error(), "google/protobuf/struct.proto") {
		t.Fatalf("got %v, want an error naming google/protobuf/struct.proto", err)
	}
}

// A dependency the server cannot find is reported with its name.
func TestReflectionServerErrorResponse(t *testing.T) {
	addr, _ := startSingleFileReflection(t, singleFileReflection{services: []string{"wombat.v1.Foobar", "no.such.Service"}})

	_, err := protoFilesFromReflectionAPI(loadCtx(t), dialTest(t, addr))
	if err == nil || !strings.Contains(err.Error(), "no.such.Service") {
		t.Fatalf("got %v, want an error naming no.such.Service", err)
	}
}
