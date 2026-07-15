package pod

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/storage"
)

// buildOnCloudBuild builds and pushes the per-project agent image without a local
// Docker daemon (ADR-0009): it uploads the build context to GCS and runs the same
// two-stage build on Cloud Build. Same Dockerfiles, same layering as the old local
// `docker build` — a different executor, which is the whole point of moving off the
// laptop. Blocks until the build finishes, so a caller's cache check followed by
// this reads like the local path it replaced.
func buildOnCloudBuild(ctx context.Context, gcpProject, bucket, projectDir, agentDir, ref string) error {
	_, tag, err := splitRef(ref)
	if err != nil {
		return err
	}
	object := fmt.Sprintf("builds/%s/source.tar.gz", tag)

	if err := uploadBuildContext(ctx, bucket, object, projectDir, agentDir); err != nil {
		return fmt.Errorf("upload build context: %w", err)
	}

	client, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("cloud build client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Two docker steps in one build share a daemon, so the env image built in the
	// first step is the base the second layers on — exactly as it worked locally.
	// The env image is never in `Images`, so it is used and discarded, never pushed.
	build := &cloudbuildpb.Build{
		Source: &cloudbuildpb.Source{
			Source: &cloudbuildpb.Source_StorageSource{
				StorageSource: &cloudbuildpb.StorageSource{Bucket: bucket, Object: object},
			},
		},
		Steps: []*cloudbuildpb.BuildStep{
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: []string{"build", "--target", "env", "-t", ProjectEnvImage, "project"},
			},
			{
				Name: "gcr.io/cloud-builders/docker",
				Args: []string{"build", "--build-arg", "BASE_IMAGE=" + ProjectEnvImage, "-t", ref, "agent"},
			},
		},
		Images: []string{ref},
	}

	op, err := client.CreateBuild(ctx, &cloudbuildpb.CreateBuildRequest{
		ProjectId: gcpProject,
		Build:     build,
	})
	if err != nil {
		return fmt.Errorf("start build: %w", err)
	}

	done, err := op.Wait(ctx)
	if err != nil {
		return fmt.Errorf("build of %s failed (see its Cloud Build log): %w", ref, err)
	}
	if done.GetStatus() != cloudbuildpb.Build_SUCCESS {
		return fmt.Errorf("build of %s finished %s (see its Cloud Build log)", ref, done.GetStatus())
	}
	return nil
}

// uploadBuildContext ships the two build inputs to GCS as one gzipped tar, each
// under its own top-level directory so the build steps can address them as
// `project` and `agent`.
func uploadBuildContext(ctx context.Context, bucket, object, projectDir, agentDir string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage client: %w", err)
	}
	defer func() { _ = client.Close() }()

	w := client.Bucket(bucket).Object(object).NewWriter(ctx)
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)

	err = firstErr(
		tarTree(tw, projectDir, "project"),
		tarTree(tw, agentDir, "agent"),
		tw.Close(),
		gz.Close(),
	)
	if err != nil {
		_ = w.Close()
		return err
	}
	// The upload is only committed on Close, so its error is the one that matters.
	if err := w.Close(); err != nil {
		return fmt.Errorf("commit %s: %w", object, err)
	}
	return nil
}

// tarTree writes every regular file and directory under dir into tw, rooted at
// prefix. Symlinks and devices are dropped: a build context has no business
// carrying them, and following one could escape the tree.
func tarTree(tw *tar.Writer, dir, prefix string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() && !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(prefix, rel))
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = io.Copy(tw, f)
		return err
	})
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
