package sandbox

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const (
	labelKey   = "ai.orchestrator"
	labelValue = "true"

	// Image is the sandbox's base image: a Go toolchain so the mounted
	// workspace can be built and tested natively.
	Image = "golang:1.23-bookworm"

	// WorkspaceRoot is the fixed, hardcoded path inside the container where
	// the workspace is mounted. All tool paths resolve relative to this.
	WorkspaceRoot = "/workspace"
)

type DockerSandbox struct {
	cli         *client.Client
	containerID string
}

func NewDockerSandbox(cli *client.Client) *DockerSandbox {
	return &DockerSandbox{cli: cli}
}

// demoProjectHostPath returns the absolute host path of the demo-project
// directory, resolved relative to this source file so it works regardless of
// the process's working directory.
func demoProjectHostPath() string {
	_, thisFile, _, _ := runtime.Caller(0)
	// this file: platform/orchestrator/internal/sandbox/docker.go
	// three levels up is the monorepo root (platform/), which holds demo-project.
	monorepoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(monorepoRoot, "demo-project")
}

// buildMounts returns the sandbox's fixed bind mount: the demo project on the
// host, mounted at the workspace root inside the container. Stands in for
// real repo-cloning until that exists.
func buildMounts() []mount.Mount {
	return []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: demoProjectHostPath(),
			Target: WorkspaceRoot,
		},
	}
}

// SweepOrphans removes any containers tagged with the orchestrator label that
// were left running by a previous crash or hard kill. Must be called before Start.
func (s *DockerSandbox) SweepOrphans(ctx context.Context) error {
	args := filters.NewArgs(filters.Arg("label", labelKey+"="+labelValue))
	orphans, err := s.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return fmt.Errorf("sandbox: list orphans: %w", err)
	}
	for _, c := range orphans {
		log.Printf("sandbox: removing orphan container %s", c.ID[:12])
		if err := s.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
			if !client.IsErrNotFound(err) {
				return fmt.Errorf("sandbox: remove orphan %s: %w", c.ID[:12], err)
			}
		}
	}
	return nil
}

func (s *DockerSandbox) Start(ctx context.Context) error {
	log.Printf("sandbox: pulling image %s", Image)
	rc, err := s.cli.ImagePull(ctx, Image, dockerimage.PullOptions{})
	if err != nil {
		return fmt.Errorf("sandbox: pull image %s: %w", Image, err)
	}
	io.Copy(io.Discard, rc)
	rc.Close()

	resp, err := s.cli.ContainerCreate(ctx,
		&container.Config{
			Image:  Image,
			Labels: map[string]string{labelKey: labelValue},
			Cmd:    []string{"sleep", "infinity"},
		},
		&container.HostConfig{
			AutoRemove: true,
			Mounts:     buildMounts(),
		},
		nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("sandbox: create container: %w", err)
	}
	s.containerID = resp.ID

	if err := s.cli.ContainerStart(ctx, s.containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("sandbox: start container: %w", err)
	}
	log.Printf("sandbox: started container %s", s.containerID[:12])
	return nil
}

func (s *DockerSandbox) Destroy(ctx context.Context) error {
	if s.containerID == "" {
		return nil
	}

	// If the caller's context is already cancelled (e.g. signal shutdown),
	// use a fresh context so the cleanup call still reaches the Docker daemon.
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	id := s.containerID
	s.containerID = ""
	log.Printf("sandbox: destroying container %s", id[:12])
	err := s.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	if err != nil {
		if client.IsErrNotFound(err) {
			// Already gone — AutoRemove fired or container was removed externally.
			log.Printf("sandbox: container %s already gone", id[:12])
			return nil
		}
		return fmt.Errorf("sandbox: destroy container %s: %w", id[:12], err)
	}
	log.Printf("sandbox: container %s destroyed", id[:12])
	return nil
}
