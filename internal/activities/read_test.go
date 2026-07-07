package activities

import (
	"context"
	"errors"
	"testing"
)

type fakeFileReader struct {
	gotPath string
	content []byte
	err     error
}

func (f *fakeFileReader) ReadFile(_ context.Context, path string) ([]byte, error) {
	f.gotPath = path
	return f.content, f.err
}

func TestRead_ReturnsFileContentResolvedAgainstWorkspaceRoot(t *testing.T) {
	fr := &fakeFileReader{content: []byte("package main\n")}

	out, err := Read(context.Background(), fr, ReadInput{Path: "main.go"})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if out.Content != "package main\n" {
		t.Fatalf("Content = %q, want %q", out.Content, "package main\n")
	}
	if fr.gotPath != WorkspaceRoot+"/main.go" {
		t.Fatalf("ReadFile called with %q, want %q", fr.gotPath, WorkspaceRoot+"/main.go")
	}
}

func TestRead_RejectsPathTraversal(t *testing.T) {
	fr := &fakeFileReader{content: []byte("should not be reached")}

	_, err := Read(context.Background(), fr, ReadInput{Path: "../../etc/passwd"})
	if err == nil {
		t.Fatal("Read: want error for traversal path, got nil")
	}
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("Read err = %v, want wrapping ErrPathTraversal", err)
	}
	if fr.gotPath != "" {
		t.Fatalf("ReadFile should not have been called, got path %q", fr.gotPath)
	}
}

func TestRead_NotFoundIsGenuineError(t *testing.T) {
	fr := &fakeFileReader{err: errors.New("no such file")}

	_, err := Read(context.Background(), fr, ReadInput{Path: "missing.go"})
	if err == nil {
		t.Fatal("Read: want error for missing file, got nil")
	}
}
