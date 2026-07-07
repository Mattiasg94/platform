package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// execRaw runs cmd inside the sandbox container via the Docker exec API and
// returns its captured stdout/stderr and exit code. A non-nil error here
// means the exec itself could not be run (plumbing failure) — it is
// distinct from the command's own exit code, which is returned as data.
// [LEARN]: This is the single chokepoint all three tools funnel through.
// ReadFile, WriteFile, and Exec are just different argv + stdin combos over
// this one Docker exec mechanism — the sandbox package has no concept of
// "read" or "write", only "run this command, capture the output".
func (s *DockerSandbox) execRaw(ctx context.Context, cmd []string, workdir string, stdin []byte) (stdout, stderr []byte, exitCode int, err error) {
	// [LEARN]: AttachStdin toggles on whether stdin is non-nil — that single
	// bool is what distinguishes a Read call (stdin=nil, just `cat path`) from
	// a Write call (stdin=file content, piped into `cat > path`) further down.
	created, err := s.cli.ContainerExecCreate(ctx, s.containerID, container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   workdir,
		AttachStdin:  stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("sandbox: exec create: %w", err)
	}

	// [LEARN]: This is the actual hijacked connection into the container's
	// exec process — a raw pipe over the Docker API. If the daemon or
	// container is unreachable, it fails *here*, which is exactly the
	// "genuine tool-execution failure" the PRD wants kept distinct from a
	// command's own non-zero exit (that comes later, from ContainerExecInspect).
	attached, err := s.cli.ContainerExecAttach(ctx, created.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, nil, 0, fmt.Errorf("sandbox: exec attach: %w", err)
	}
	defer attached.Close()

	if stdin != nil {
		if _, err := attached.Conn.Write(stdin); err != nil {
			return nil, nil, 0, fmt.Errorf("sandbox: exec write stdin: %w", err)
		}
		// [LEARN]: Without this, `cat > path` (used by WriteFile) would block
		// forever waiting for more stdin — CloseWrite sends EOF on the pipe so
		// cat knows the file content is complete and can exit.
		attached.CloseWrite()
	}

	// [LEARN]: Docker multiplexes stdout and stderr over this one connection
	// when there's no TTY, framing each chunk with a header byte saying which
	// stream it belongs to. stdcopy.StdCopy is what demuxes that back into
	// two separate byte streams — without it you'd get one interleaved blob.
	var outBuf, errBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&outBuf, &errBuf, attached.Reader); err != nil {
		// A context timeout surfaces here as the attach connection closing
		// mid-read; report it distinctly rather than as a generic I/O error.
		if ctx.Err() != nil {
			return nil, nil, 0, fmt.Errorf("sandbox: exec %v: %w", cmd, ctx.Err())
		}
		return nil, nil, 0, fmt.Errorf("sandbox: exec read output: %w", err)
	}

	// [LEARN]: Exit code isn't part of the attach stream at all — it only
	// exists once the process has exited, so it needs this separate inspect
	// call. StdCopy above only returns once the process's stdout/stderr have
	// hit EOF (i.e. it has exited), which is what makes calling Inspect here safe.
	inspect, err := s.cli.ContainerExecInspect(ctx, created.ID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("sandbox: exec inspect: %w", err)
	}

	// [LEARN]: This is the seam the whole "non-zero exit is data, not a Go
	// error" contract hinges on. execRaw's err return is reserved for the
	// plumbing calls above (create/attach/inspect) — a failing test run's
	// exit code flows out here as an ordinary value, never as `err`.
	return outBuf.Bytes(), errBuf.Bytes(), inspect.ExitCode, nil
}

// ReadFile returns a file's contents from the workspace. A nonexistent file
// (or any other read failure) is a genuine error.
func (s *DockerSandbox) ReadFile(ctx context.Context, path string) ([]byte, error) {
	stdout, stderr, exitCode, err := s.execRaw(ctx, []string{"cat", path}, WorkspaceRoot, nil)
	if err != nil {
		return nil, err
	}
	// [LEARN]: Here's the flip side of the note in execRaw — the *same*
	// exec plumbing is reused, but this call site chooses to promote a
	// non-zero exit (cat failing because the file's missing) into a Go
	// error. That's a deliberate per-caller decision, not something
	// execRaw enforces: Exec (below) makes the opposite choice for the
	// same shape of result.
	if exitCode != 0 {
		return nil, fmt.Errorf("sandbox: read %q: %s", path, bytes.TrimSpace(stderr))
	}
	return stdout, nil
}

// WriteFile creates or overwrites a file in the workspace, auto-creating any
// missing parent directories. Content is passed over stdin (not interpolated
// into the shell command) so it can contain arbitrary bytes safely.
func (s *DockerSandbox) WriteFile(ctx context.Context, path string, content []byte) error {
	// "$1" is a positional arg, not string-interpolated, so path can't break
	// out of the shell command even though it comes from model-supplied input.
	// [LEARN]: This function never re-validates path itself — by the time a
	// path reaches here it's already been through resolvePath in the
	// activities package (workspace-root join + traversal rejection). This
	// layer trusts its caller; the guard lives one level up.
	cmd := []string{"sh", "-c", `mkdir -p "$(dirname "$1")" && cat > "$1"`, "sh", path}
	_, stderr, exitCode, err := s.execRaw(ctx, cmd, WorkspaceRoot, content)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("sandbox: write %q: %s", path, bytes.TrimSpace(stderr))
	}
	return nil
}

// Exec runs cmd in the sandbox's persistent shell, so `cd` and exported env
// vars from one call are visible to the next — the way a real terminal
// session behaves. A non-zero exit code is ordinary result data, not a
// Go-level error.
//
// [LEARN]: Known gap, deliberately deferred to the idle-timeout issue: a
// hung command blocks this call forever. There is no kill here yet — only
// detecting and killing a stuck process group is what that issue adds.
func (s *DockerSandbox) Exec(ctx context.Context, cmd []string, workdir string) (ExecResult, error) {
	if workdir == "" {
		workdir = WorkspaceRoot
	}

	s.shellMu.Lock()
	defer s.shellMu.Unlock()

	marker, err := newMarker()
	if err != nil {
		return ExecResult{}, err
	}

	// Only cd when the caller explicitly asked for a different workdir than
	// last time — otherwise a call that leaves workdir at its default would
	// stomp a `cd` the model just ran as an ordinary command.
	cdTo := ""
	if workdir != s.shellLastWorkdir {
		cdTo = workdir
		s.shellLastWorkdir = workdir
	}

	line := buildCommandLine(cmd, marker, cdTo)
	if _, err := io.WriteString(s.shell.Conn, line); err != nil {
		return ExecResult{}, fmt.Errorf("sandbox: write to shell: %w", err)
	}

	stdout, stderr, exitCode, err := demuxUntilMarker(s.shell.Reader, marker)
	if err != nil {
		return ExecResult{}, fmt.Errorf("sandbox: exec %v: %w", cmd, err)
	}

	// [LEARN]: This ExecResult is what eventually becomes the genai
	// FunctionResponse handed back to the model — stdout/stderr/exitCode
	// round-trip all the way from this Docker call up through
	// activities.Exec and the tools.Registry adapter to the LLM as plain
	// data, letting it "read" a failing test run the way a person would.
	return ExecResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}
