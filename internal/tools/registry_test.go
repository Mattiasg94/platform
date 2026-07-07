package tools

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/sandbox"
)

var errNotReachable = errors.New("sandbox unreachable")

type fakeSandbox struct {
	readPath string
	content  []byte
	readErr  error

	writePath    string
	writeContent []byte
	writeErr     error

	execCmd     []string
	execWorkdir string
	execResult  sandbox.ExecResult
	execErr     error
}

func (f *fakeSandbox) ReadFile(_ context.Context, path string) ([]byte, error) {
	f.readPath = path
	return f.content, f.readErr
}

func (f *fakeSandbox) WriteFile(_ context.Context, path string, content []byte) error {
	f.writePath = path
	f.writeContent = content
	return f.writeErr
}

func (f *fakeSandbox) Exec(_ context.Context, cmd []string, workdir string) (sandbox.ExecResult, error) {
	f.execCmd = cmd
	f.execWorkdir = workdir
	return f.execResult, f.execErr
}

func TestExecute_ReadFile_ReturnsContent(t *testing.T) {
	fs := &fakeSandbox{content: []byte("hello")}
	r := NewRegistry(fs)

	got, err := r.Execute(context.Background(), "read_file", map[string]any{"path": "main.go"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got["content"] != "hello" {
		t.Fatalf("content = %v, want %q", got["content"], "hello")
	}
}

func TestExecute_ReadFile_MissingPathArgIsError(t *testing.T) {
	fs := &fakeSandbox{}
	r := NewRegistry(fs)

	_, err := r.Execute(context.Background(), "read_file", map[string]any{})
	if err == nil {
		t.Fatal("Execute: want error for missing path arg, got nil")
	}
}

func TestExecute_WriteFile_WritesContent(t *testing.T) {
	fs := &fakeSandbox{}
	r := NewRegistry(fs)

	_, err := r.Execute(context.Background(), "write_file", map[string]any{"path": "fib.go", "content": "package main\n"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if string(fs.writeContent) != "package main\n" {
		t.Fatalf("write content = %q", fs.writeContent)
	}
}

func TestExecute_WriteFile_MissingArgsIsError(t *testing.T) {
	fs := &fakeSandbox{}
	r := NewRegistry(fs)

	_, err := r.Execute(context.Background(), "write_file", map[string]any{"path": "fib.go"})
	if err == nil {
		t.Fatal("Execute: want error for missing content arg, got nil")
	}
}

func TestExecute_Bash_ReturnsStdoutStderrExitCode(t *testing.T) {
	fs := &fakeSandbox{execResult: sandbox.ExecResult{Stdout: "FAIL\n", Stderr: "", ExitCode: 1}}
	r := NewRegistry(fs)

	got, err := r.Execute(context.Background(), "bash", map[string]any{
		"command": []any{"go", "test", "./..."},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got["stdout"] != "FAIL\n" {
		t.Fatalf("stdout = %v", got["stdout"])
	}
	if got["exit_code"] != 1 {
		t.Fatalf("exit_code = %v, want 1", got["exit_code"])
	}
	if fs.execCmd[0] != "go" || fs.execCmd[1] != "test" || fs.execCmd[2] != "./..." {
		t.Fatalf("execCmd = %v", fs.execCmd)
	}
}

func TestExecute_Bash_MissingCommandIsError(t *testing.T) {
	fs := &fakeSandbox{}
	r := NewRegistry(fs)

	_, err := r.Execute(context.Background(), "bash", map[string]any{})
	if err == nil {
		t.Fatal("Execute: want error for missing command arg, got nil")
	}
}

func TestExecute_Bash_PlumbingFailureIsError(t *testing.T) {
	fs := &fakeSandbox{execErr: errNotReachable}
	r := NewRegistry(fs)

	_, err := r.Execute(context.Background(), "bash", map[string]any{"command": []any{"ls"}})
	if err == nil {
		t.Fatal("Execute: want error for plumbing failure, got nil")
	}
}

func TestExecute_UnknownTool_ReturnsError(t *testing.T) {
	r := NewRegistry(&fakeSandbox{})

	_, err := r.Execute(context.Background(), "does_not_exist", map[string]any{})
	if err == nil {
		t.Fatal("Execute: want error for unknown tool, got nil")
	}
}
