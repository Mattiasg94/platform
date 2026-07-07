package activities

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/sandbox"
)

type fakeCommandExecutor struct {
	gotCmd     []string
	gotWorkdir string
	result     sandbox.ExecResult
	err        error
}

func (f *fakeCommandExecutor) Exec(_ context.Context, cmd []string, workdir string) (sandbox.ExecResult, error) {
	f.gotCmd = cmd
	f.gotWorkdir = workdir
	return f.result, f.err
}

func TestExec_DefaultsWorkdirToWorkspaceRoot(t *testing.T) {
	ce := &fakeCommandExecutor{result: sandbox.ExecResult{Stdout: "ok", ExitCode: 0}}

	out, err := Exec(context.Background(), ce, ExecInput{Command: []string{"go", "test", "./..."}})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if ce.gotWorkdir != WorkspaceRoot {
		t.Fatalf("workdir = %q, want %q", ce.gotWorkdir, WorkspaceRoot)
	}
	if out.Stdout != "ok" || out.ExitCode != 0 {
		t.Fatalf("out = %+v", out)
	}
}

func TestExec_ResolvesDirRelativeToWorkspaceRoot(t *testing.T) {
	ce := &fakeCommandExecutor{}

	_, err := Exec(context.Background(), ce, ExecInput{Command: []string{"ls"}, Dir: "sub/pkg"})
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if ce.gotWorkdir != WorkspaceRoot+"/sub/pkg" {
		t.Fatalf("workdir = %q, want %q", ce.gotWorkdir, WorkspaceRoot+"/sub/pkg")
	}
}

func TestExec_RejectsDirTraversal(t *testing.T) {
	ce := &fakeCommandExecutor{}

	_, err := Exec(context.Background(), ce, ExecInput{Command: []string{"ls"}, Dir: "../outside"})
	if !errors.Is(err, ErrPathTraversal) {
		t.Fatalf("Exec err = %v, want wrapping ErrPathTraversal", err)
	}
	if ce.gotCmd != nil {
		t.Fatal("Exec should not have been called on the executor")
	}
}

func TestExec_EmptyCommandIsError(t *testing.T) {
	ce := &fakeCommandExecutor{}

	_, err := Exec(context.Background(), ce, ExecInput{Command: nil})
	if err == nil {
		t.Fatal("Exec: want error for empty command, got nil")
	}
}

func TestExec_NonZeroExitCodeIsNotAGoError(t *testing.T) {
	ce := &fakeCommandExecutor{result: sandbox.ExecResult{Stdout: "FAIL", ExitCode: 1}}

	out, err := Exec(context.Background(), ce, ExecInput{Command: []string{"go", "test"}})
	if err != nil {
		t.Fatalf("Exec: want nil error for a command's own failing exit code, got %v", err)
	}
	if out.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", out.ExitCode)
	}
}

func TestExec_PlumbingFailureIsAGoError(t *testing.T) {
	ce := &fakeCommandExecutor{err: errors.New("sandbox unreachable")}

	_, err := Exec(context.Background(), ce, ExecInput{Command: []string{"go", "test"}})
	if err == nil {
		t.Fatal("Exec: want error for plumbing failure, got nil")
	}
}
