package app

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
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

	client := grpcreflect.NewClientAuto(ctx, conn)
	defer client.Reset()

	services, err := client.ListServices()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	fdset := &descriptorpb.FileDescriptorSet{}

	for _, srv := range services {
		fd, err := client.FileContainingSymbol(srv)
		if err != nil {
			return nil, err
		}

		// Add file descriptor and its dependencies to the set
		collectFileDescriptors(seen, fdset, fd)
	}

	return protodesc.NewFiles(fdset)
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

// collectFileDescriptors adds file descriptors to the set
func collectFileDescriptors(seen map[string]struct{}, fdset *descriptorpb.FileDescriptorSet, fd *desc.FileDescriptor) {
	if fd == nil {
		return
	}

	fdProto := fd.AsFileDescriptorProto()
	if fdProto == nil || fdProto.GetName() == "" {
		return
	}

	if _, ok := seen[fdProto.GetName()]; ok {
		return
	}

	seen[fdProto.GetName()] = struct{}{}
	fdset.File = append(fdset.File, fdProto)

	// Process dependencies
	for _, dep := range fd.GetDependencies() {
		collectFileDescriptors(seen, fdset, dep)
	}
}

