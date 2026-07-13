package pod

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ProjectEnvImage is a local build stage, never pushed: it is only the base the
// agent image is layered on (ADR-0009).
const ProjectEnvImage = "demo-project-env"

// Builder turns the project's Dockerfile and the agent's Dockerfile into a
// published image, and does nothing else. It is the one part of the platform that
// still needs a Docker daemon, which is why it stays on the laptop while the agent
// itself runs in the cloud. When the orchestrator moves to Cloud Run it loses that
// daemon, and this becomes a Cloud Build trigger — same shape, different executor.
type Builder struct {
	projectDir string
	gcpProject string

	// image is the content-addressed ref EnsureImage resolved.
	image string
}

func NewBuilder(projectDir, gcpProject string) *Builder {
	return &Builder{projectDir: projectDir, gcpProject: gcpProject}
}

// EnsureImage guarantees the image this run needs exists in the registry — usually
// by discovering it already does. The environment changes far more slowly than the
// code, so the overwhelmingly common case is a cache hit and no build at all. A
// miss means someone edited a Dockerfile, and the rebuild happens by itself.
func (b *Builder) EnsureImage(ctx context.Context) error {
	ref, err := b.imageRef()
	if err != nil {
		return err
	}
	b.image = ref

	exists, err := imageExists(ctx, ref)
	if err != nil {
		return err
	}
	if exists {
		log.Printf("image %s — cached", ref)
		return nil
	}

	log.Printf("image %s — not in the registry, building", ref)
	return b.buildAndPush(ctx, ref)
}

func (b *Builder) buildAndPush(ctx context.Context, ref string) error {
	// --target env is the source-free toolchain stage; editing the workspace must
	// not invalidate the agent image layered on top of it.
	if err := dockerBuild(ctx, ProjectEnvImage, b.projectDir, "--target", "env"); err != nil {
		return fmt.Errorf("build project env image: %w", err)
	}
	if err := dockerBuild(ctx, ref, agentHostPath(), "--build-arg", "BASE_IMAGE="+ProjectEnvImage); err != nil {
		return fmt.Errorf("build agent image: %w", err)
	}
	return dockerPush(ctx, ref)
}

func dockerBuild(ctx context.Context, tag, contextDir string, extra ...string) error {
	args := append([]string{"build", "-t", tag}, extra...)
	args = append(args, contextDir)
	return docker(ctx, args...)
}

func dockerPush(ctx context.Context, ref string) error {
	return docker(ctx, "push", ref)
}

func docker(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	// Capture the chatter and surface it only on failure.
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s: %w\n%s", args[0], err, out.String())
	}
	return nil
}

func agentHostPath() string {
	return filepath.Join(monorepoRoot(), "agent")
}

func monorepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}
