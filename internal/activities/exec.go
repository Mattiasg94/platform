package activities

import (
	"context"
	"fmt"

	"orchestrator/internal/sandbox"
)

// CommandExecutor is the sandbox capability Exec depends on.
type CommandExecutor interface {
	Exec(ctx context.Context, cmd []string, workdir string) (sandbox.ExecResult, error)
}

type ExecInput struct {
	Command []string
	// Dir is optional, relative to the workspace root; defaults to the root.
	Dir string
}

type ExecOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec runs an arbitrary command inside the sandbox workspace. A non-zero
// exit code is ordinary result data, not a Go-level error — only a genuine
// execution-plumbing failure (sandbox unreachable, exec API failure) is.
func Exec(ctx context.Context, e CommandExecutor, in ExecInput) (ExecOutput, error) {
	if len(in.Command) == 0 {
		return ExecOutput{}, fmt.Errorf("activities: empty command")
	}

	workdir := WorkspaceRoot
	if in.Dir != "" {
		full, err := resolvePath(in.Dir)
		if err != nil {
			return ExecOutput{}, err
		}
		workdir = full
	}

	res, err := e.Exec(ctx, in.Command, workdir)
	if err != nil {
		return ExecOutput{}, fmt.Errorf("activities: exec %v: %w", in.Command, err)
	}
	return ExecOutput{
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		ExitCode: res.ExitCode,
	}, nil
}
