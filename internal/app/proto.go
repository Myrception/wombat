package app

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ctxInternalKey struct{}

func protoFilesFromReflectionAPI(ctx context.Context, conn *grpc.ClientConn) (*protoregistry.Files, error) {
	if conn == nil {
		return nil, errors.New("app: no connection to a grpc server available")
	}

	// Create reflection client
	client := grpc_reflection_v1.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.CloseSend()

	// Get list of all services
	err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	})
	if err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return nil, errors.New("app: invalid reflection response")
	}

	// Process each service to get its file descriptor
	seen := make(map[string]struct{})
	fdset := &descriptorpb.FileDescriptorSet{}

	for _, service := range listResp.Service {
		// Request file descriptor for this service
		err = stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
			MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: service.Name,
			},
		})
		if err != nil {
			return nil, err
		}

		resp, err = stream.Recv()
		if err != nil {
			return nil, err
		}

		fdResp := resp.GetFileDescriptorResponse()
		if fdResp == nil {
			return nil, errors.New("app: invalid file descriptor response")
		}

		// Process the file descriptor and its dependencies
		for _, fdBytes := range fdResp.FileDescriptorProto {
			fd := &descriptorpb.FileDescriptorProto{}
			if err := proto.Unmarshal(fdBytes, fd); err != nil {
				return nil, err
			}
			addFileDescriptor(seen, fdset, fd, stream, ctx)
		}
	}

	return protodesc.NewFiles(fdset)
}

// addFileDescriptor adds a file descriptor and its dependencies to the set
func addFileDescriptor(seen map[string]struct{}, fdset *descriptorpb.FileDescriptorSet, fd *descriptorpb.FileDescriptorProto, stream grpc_reflection_v1.ServerReflection_ServerReflectionInfoClient, ctx context.Context) error {
	if fd == nil || fd.GetName() == "" {
		return nil
	}

	if _, ok := seen[fd.GetName()]; ok {
		return nil
	}

	seen[fd.GetName()] = struct{}{}
	fdset.File = append(fdset.File, fd)

	// Fetch dependencies recursively
	for _, dep := range fd.GetDependency() {
		if _, ok := seen[dep]; ok {
			continue
		}

		err := stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
			MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_FileByFilename{
				FileByFilename: dep,
			},
		})
		if err != nil {
			return err
		}

		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		fdResp := resp.GetFileDescriptorResponse()
		if fdResp == nil {
			return errors.New("app: invalid file descriptor response for dependency")
		}

		for _, depBytes := range fdResp.FileDescriptorProto {
			depFd := &descriptorpb.FileDescriptorProto{}
			if err := proto.Unmarshal(depBytes, depFd); err != nil {
				return err
			}
			if err := addFileDescriptor(seen, fdset, depFd, stream, ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// protoFilesFromDisk uses protoc to compile proto files into a descriptor set
func protoFilesFromDisk(importPaths, filenames []string) (*protoregistry.Files, error) {
	if len(filenames) == 0 {
		return nil, errors.New("app: no *.proto files found")
	}

	// Create temporary file for the descriptor set
	tempFile, err := os.CreateTemp("", "descriptor_set_*.pb")
	if err != nil {
		return nil, err
	}
	tempFilePath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempFilePath)

	// Build protoc command with import paths and output
	args := []string{"--descriptor_set_out=" + tempFilePath, "--include_imports"}

	// Add import paths
	for _, importPath := range importPaths {
		args = append(args, "--proto_path="+importPath)
	}

	// Add filenames
	args = append(args, filenames...)

	// Run protoc
	cmd := exec.Command("protoc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New("app: protoc execution failed: " + string(output))
	}

	// Read the generated descriptor set
	descSetBytes, err := os.ReadFile(tempFilePath)
	if err != nil {
		return nil, err
	}

	// Parse the descriptor set
	fdset := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(descSetBytes, fdset); err != nil {
		return nil, err
	}

	return protodesc.NewFiles(fdset)
}

