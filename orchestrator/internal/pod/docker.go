package pod

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	Image           = "agent-pod"
	ProjectEnvImage = "demo-project-env"
	VerifyImage     = "demo-project-verify"
	WorkspaceRoot   = "/workspace"
)

var _ Runner = (*Docker)(nil)

type Docker struct {
	cli        *client.Client
	projectDir string
}

// NewDocker binds the runner to projectDir — the checked-out project tree the
// agent edits. It is the build context for the env/verify images and the source
// of the workspace bind mount, so one runner is scoped to one checkout.
func NewDocker(projectDir string) (*Docker, error) {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	// The Go SDK reads DOCKER_HOST but not docker CLI contexts, so when it's unset
	// resolve the active context's endpoint ourselves.
	if os.Getenv("DOCKER_HOST") == "" {
		if host := resolveDockerHost(); host != "" {
			opts = append(opts, client.WithHost(host))
		}
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &Docker{cli: cli, projectDir: projectDir}, nil
}

func (d *Docker) EnsureImage(ctx context.Context) error {
	// --target env is the source-free toolchain stage; editing the workspace must
	// not invalidate the agent image layered on top of it.
	if err := timed("build project env image ("+ProjectEnvImage+")", func() error {
		return dockerBuild(ctx, ProjectEnvImage, d.projectDir, "--target", "env")
	}); err != nil {
		return fmt.Errorf("build project env image: %w", err)
	}
	if err := timed("build agent image ("+Image+")", func() error {
		return dockerBuild(ctx, Image, agentHostPath(), "--build-arg", "BASE_IMAGE="+ProjectEnvImage)
	}); err != nil {
		return fmt.Errorf("build agent image: %w", err)
	}
	return nil
}

func timed(stage string, fn func() error) error {
	log.Printf("%s…", stage)
	start := time.Now()
	err := fn()
	log.Printf("%s — %s", stage, time.Since(start).Round(time.Millisecond))
	return err
}

func dockerBuild(ctx context.Context, tag, contextDir string, extra ...string) error {
	args := append([]string{"build", "-t", tag}, extra...)
	args = append(args, contextDir)
	cmd := exec.CommandContext(ctx, "docker", args...)
	// Capture build chatter and surface it only on failure; on success the
	// timed() line is signal enough.
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w\n%s", err, out.String())
	}
	return nil
}

type Verification struct {
	Passed bool
	Output string
}

// Verify re-runs the suite in a fresh container the agent never touched, so the
// verdict can't be gamed (ADR-0005). A non-zero test exit is Passed=false, not an
// error; only failing to run the container at all is an error.
func (d *Docker) Verify(ctx context.Context) (Verification, error) {
	var v Verification
	err := timed("verification", func() error {
		var err error
		v, err = d.verify(ctx)
		return err
	})
	return v, err
}

func (d *Docker) verify(ctx context.Context) (Verification, error) {
	if err := timed("build verify image ("+VerifyImage+")", func() error {
		return dockerBuild(ctx, VerifyImage, d.projectDir, "--target", "verify")
	}); err != nil {
		return Verification{}, fmt.Errorf("build verify image: %w", err)
	}

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", VerifyImage)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return Verification{}, fmt.Errorf("run verify image: %w", err)
	}
	return Verification{Passed: err == nil, Output: out.String()}, nil
}

func resolveDockerHost() string {
	out, err := exec.Command("docker", "context", "inspect", "-f", "{{.Endpoints.docker.Host}}").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (d *Docker) Run(ctx context.Context, prompt string) (Result, error) {
	var result Result
	err := timed("pod run", func() error {
		var err error
		result, err = d.runPod(ctx, prompt)
		return err
	})
	return result, err
}

func (d *Docker) runPod(ctx context.Context, prompt string) (Result, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return Result{}, fmt.Errorf("ANTHROPIC_API_KEY not set (checked .env and host env)")
	}

	created, err := d.cli.ContainerCreate(ctx,
		&container.Config{
			Image:      Image,
			Cmd:        []string{prompt},
			Env:        []string{"ANTHROPIC_API_KEY=" + apiKey, "HOME=/tmp"},
			User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
			WorkingDir: WorkspaceRoot,
		},
		&container.HostConfig{
			Mounts: []mount.Mount{{
				Type:   mount.TypeBind,
				Source: d.projectDir,
				Target: WorkspaceRoot,
			}},
		},
		nil, nil, "",
	)
	if err != nil {
		return Result{}, fmt.Errorf("create pod: %w", err)
	}
	id := created.ID
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
		// Exit code is ignored: a non-zero exit still carries a JSON error result
		// on stdout, which is more informative.
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

func agentHostPath() string {
	return filepath.Join(monorepoRoot(), "agent")
}

func monorepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
