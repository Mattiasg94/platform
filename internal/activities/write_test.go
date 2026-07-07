package activities

import (
	"context"
	"errors"
	"testing"
)

type fakeFileWriter struct {
	gotPath    string
	gotContent []byte
	err        error
}

func (f *fakeFileWriter) WriteFile(_ context.Context, path string, content []byte) error {
	f.gotPath = path
	f.gotContent = content
	return f.err
}

func TestWrite_WritesContentResolvedAgainstWorkspaceRoot(t *testing.T) {
	fw := &fakeFileWriter{}

	_, err := Write(context.Background(), fw, WriteInput{Path: "fib.go", Content: "package main\n"})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if fw.gotPath != WorkspaceRoot+"/fib.go" {
		t.Fatalf("WriteFile called with path %q, want %q", fw.gotPath, WorkspaceRoot+"/fib.go")
	}
	if string(fw.gotContent) != "package main\n" {
		t.Fatalf("WriteFile called with content %q", fw.gotContent)
	}
}

func TestWrite_RejectsPathTraversal(t *testing.T) {
	fw := &fakeFileWriter{}

	_, err := Write(context.Background(), fw, WriteInput{Path: "../outside.go", Content: "x"})
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("Write err = %v, want wrapping ErrPathTraversal", err)
	}
	if fw.gotPath != "" {
		t.Fatalf("WriteFile should not have been called, got path %q", fw.gotPath)
	}
}

func TestWrite_PropagatesUnderlyingError(t *testing.T) {
	fw := &fakeFileWriter{err: errors.New("disk full")}

	_, err := Write(context.Background(), fw, WriteInput{Path: "fib.go", Content: "x"})
	if err == nil {
		t.Fatal("Write: want error, got nil")
	}
}
