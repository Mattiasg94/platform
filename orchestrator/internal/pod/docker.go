package pod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	// Image is the baked agent pod (built from agent/Dockerfile).
	Image = "agent-pod"
	// WorkspaceRoot is where the repo is mounted inside the pod.
	WorkspaceRoot = "/workspace"
)

// Docker runs the agent pod as a bounded job on a plain Docker container
// (ADR-0004). It creates the container, streams the task in, waits for exit,
// and reads the structured result back off stdout.
type Docker struct {
	cli *client.Client
}

func NewDocker() (*Docker, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	// The Go SDK reads DOCKER_HOST but not docker CLI *contexts*, and Docker
	// Desktop serves its own socket. When DOCKER_HOST is unset, resolve the
	// active context's endpoint so the orchestrator finds the daemon on its own
	// — no external env wiring. In the cloud DOCKER_HOST is set and this is a
	// no-op.
	if os.Getenv("DOCKER_HOST") == "" {
		if host := resolveDockerHost(); host != "" {
			opts = append(opts, client.WithHost(host))
		}
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Docker{cli: cli}, nil
}

// EnsureImage builds the agent pod image from source so the orchestrator
// initiates everything itself — no build step outside the process. Local-dev
// convenience; in the cloud this becomes a registry pull of a prebuilt, tagged
// image. The build is cached, so a no-change rebuild is near-instant.
func (d *Docker) EnsureImage(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", Image, agentHostPath())
	cmd.Stdout = os.Stderr // build chatter is trace, not result — keep it off stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build agent image: %w", err)
	}
	return nil
}

// resolveDockerHost asks the docker CLI for the active context's daemon
// endpoint (what `DOCKER_HOST` would otherwise hold).
func resolveDockerHost() string {
	out, err := exec.Command("docker", "context", "inspect", "-f", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Run launches the pod with the task as its command argument, mounts the demo
// project at the workspace root, and returns the parsed result. The pod prints
// its JSON result to stdout and its harness trace to stderr; we demux and parse
// only stdout.
func (d *Docker) Run(ctx context.Context, prompt string) (Result, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return Result{}, fmt.Errorf("ANTHROPIC_API_KEY not set (checked .env and host env)")
	}

	created, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:      Image,
			Cmd:        []string{prompt}, // the task, sourced from the orchestrator
			Env:        []string{"ANTHROPIC_API_KEY=" + apiKey, "HOME=/tmp"},
			User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
			WorkingDir: WorkspaceRoot,
		},
		&container.HostConfig{
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: demoProjectHostPath(),
				Target: WorkspaceRoot,
			}},
		},
		nil, nil, "",
	)
	if err != nil {
		return Result{}, fmt.Errorf("create pod: %w", err)
	}
	id := created.ID
	// Best-effort cleanup; the container is one-shot.
	defer func() {
		_ = d.cli.ContainerRemove(context.WithoutCancel(ctx), id, container.RemoveOptions{Force: true})
	}()

	if err := d.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return Result{}, fmt.Errorf("start pod: %w", err)
	}

	statusCh, errCh := d.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return Result{}, fmt.Errorf("wait for pod: %w", err)
		}
	case <-statusCh:
		// Exit code isn't load-bearing yet: a non-zero exit still carries a
		// JSON error result on stdout, which is more informative than the code.
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}

	logs, err := d.cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return Result{}, fmt.Errorf("read pod logs: %w", err)
	}
	defer func() { _ = logs.Close() }()

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, logs); err != nil {
		return Result{}, fmt.Errorf("demux pod logs: %w", err)
	}

	var result Result
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		return Result{}, fmt.Errorf("parse pod result (stdout=%q, stderr tail=%q): %w",
			stdout.String(), tail(stderr.String(), 500), err)
	}
	return result, nil
}

// demoProjectHostPath resolves the fixture repo the pod edits, relative to this
// source file so it is independent of the working directory. Stands in for real
// repo-cloning until that exists.
func demoProjectHostPath() string {
	return filepath.Join(monorepoRoot(), "demo-project")
}

// agentHostPath is the agent image's build context (agent/Dockerfile + main.py).
func agentHostPath() string {
	return filepath.Join(monorepoRoot(), "agent")
}

// monorepoRoot resolves the repo root relative to this source file, so paths are
// independent of the working directory the orchestrator runs from.
func monorepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// this file: platform/orchestrator/internal/pod/docker.go
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
