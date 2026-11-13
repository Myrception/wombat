package app

import (
	"errors"
	"os"
	"os/exec"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ctxInternalKey struct{}

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
	_ = tempFile.Close() // protoc writes it by name
	defer func() { _ = os.Remove(tempFilePath) }()

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
