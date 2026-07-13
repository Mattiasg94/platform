// Package runs is the blackboard: the bucket the orchestrator and the agent talk
// through. The dispatch message carries only a run id; everything with size —
// the task, the workspace, the diff — lives here (the Claim Check pattern).
//
// It is deliberately a dumb blob store. It moves bytes and knows nothing about
// what they mean; the pod's I/O contract (ADR-0007) stays in package pod.
package runs

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
)

const (
	taskObject      = "input.json"
	workspaceObject = "workspace.tar"
	resultObject    = "result.json"
)

type Store struct {
	bucket *storage.BucketHandle
}

func NewStore(ctx context.Context, bucket string) (*Store, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage client: %w", err)
	}
	return &Store{bucket: client.Bucket(bucket)}, nil
}

// PutTask writes the task the agent is to perform. It is an object rather than a
// command-line argument because prompts outgrow argv's size limit, and because
// the content of the work has no business being part of the invocation.
func (s *Store) PutTask(ctx context.Context, runID, task string) error {
	body, err := json.Marshal(map[string]string{"task": task})
	if err != nil {
		return fmt.Errorf("encode task: %w", err)
	}
	return s.put(ctx, s.object(runID, taskObject), func(w io.Writer) error {
		_, err := w.Write(body)
		return err
	})
}

// PutWorkspace ships dir to the agent as a tarball. The .git directory rides
// along, so the agent unpacks a real repository and `git diff` works there
// exactly as it did when the tree was bind-mounted.
func (s *Store) PutWorkspace(ctx context.Context, runID, dir string) error {
	return s.put(ctx, s.object(runID, workspaceObject), func(w io.Writer) error {
		return writeTar(w, dir)
	})
}

// GetResult returns the agent's result as raw bytes. Decoding it is package pod's
// job: the contract belongs with the seam that owns it, not with the transport.
func (s *Store) GetResult(ctx context.Context, runID string) ([]byte, error) {
	r, err := s.bucket.Object(s.object(runID, resultObject)).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open result: %w", err)
	}
	defer func() { _ = r.Close() }()

	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read result: %w", err)
	}
	return body, nil
}

func (s *Store) object(runID, name string) string {
	return filepath.Join("runs", runID, name)
}

func (s *Store) put(ctx context.Context, object string, write func(io.Writer) error) error {
	w := s.bucket.Object(object).NewWriter(ctx)
	if err := write(w); err != nil {
		_ = w.Close()
		return fmt.Errorf("write %s: %w", object, err)
	}
	// The upload is only committed on Close, so its error is the one that matters.
	if err := w.Close(); err != nil {
		return fmt.Errorf("commit %s: %w", object, err)
	}
	return nil
}

func writeTar(w io.Writer, dir string) error {
	tw := tar.NewWriter(w)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return addToTar(tw, dir, path, d)
	})
	if err != nil {
		_ = tw.Close()
		return fmt.Errorf("walk %s: %w", dir, err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}
	return nil
}

func addToTar(tw *tar.Writer, root, path string, d fs.DirEntry) error {
	// Symlinks and devices are dropped rather than followed: a checkout has no
	// business carrying them, and following one could escape the tree.
	if !d.Type().IsRegular() && !d.IsDir() {
		return nil
	}
	info, err := d.Info()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(rel)
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
}
