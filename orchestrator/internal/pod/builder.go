package pod

import (
	"context"
	"log"
	"path/filepath"
	"runtime"
)

// ProjectEnvImage is a build stage tag used only inside a build, never pushed: it
// is the base the agent image is layered on (ADR-0009).
const ProjectEnvImage = "demo-project-env"

// Builder turns the project's Dockerfile and the agent's Dockerfile into a
// published image, and does nothing else. The build runs on Cloud Build, not a
// local daemon (ADR-0009): the context is uploaded to GCS and built in the cloud,
// so the orchestrator needs no Docker and can itself run as a Cloud Run service.
type Builder struct {
	projectDir string
	gcpProject string
	bucket     string // where the build context is staged for Cloud Build

	// image is the content-addressed ref EnsureImage resolved.
	image string
}

func NewBuilder(projectDir, gcpProject, bucket string) *Builder {
	return &Builder{projectDir: projectDir, gcpProject: gcpProject, bucket: bucket}
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

	log.Printf("image %s — not in the registry, building on Cloud Build", ref)
	return buildOnCloudBuild(ctx, b.gcpProject, b.bucket, b.projectDir, agentHostPath(), ref)
}

func agentHostPath() string {
	return filepath.Join(monorepoRoot(), "agent")
}

func monorepoRoot() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
}
