package sandbox

import "context"

type Sandbox interface {
	Start(ctx context.Context) error
	Destroy(ctx context.Context) error
}

// ExecResult is the outcome of running a command in the sandbox: stdout,
// stderr, and exit code as ordinary data. A non-zero ExitCode is not an
// error — it's the command's own outcome, same as a developer reading a
// failing terminal command.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
