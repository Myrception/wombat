package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestFindProtoFiles(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"a.proto", "sub/b.proto", "sub/notes.txt", "sub/deep/c.proto"} {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// An unreadable subdirectory is skipped, not fatal.
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	files, err := findProtoFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	want := []string{filepath.Join(dir, "a.proto"), filepath.Join(dir, "sub/b.proto"), filepath.Join(dir, "sub/deep/c.proto")}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", files, want)
	}
}

func TestFindProtoFilesErrors(t *testing.T) {
	if _, err := findProtoFiles(t.TempDir()); err == nil || !strings.Contains(err.Error(), "no *.proto files found") {
		t.Errorf("empty dir: got %v", err)
	}
	if _, err := findProtoFiles(filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "failed to search") {
		t.Errorf("missing dir: got %v", err)
	}
}
