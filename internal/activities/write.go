package activities

import (
	"context"
	"fmt"
)

// FileWriter is the sandbox capability Write depends on.
type FileWriter interface {
	WriteFile(ctx context.Context, path string, content []byte) error
}

type WriteInput struct {
	Path    string
	Content string
}

type WriteOutput struct{}

// Write creates or overwrites a file's contents in the sandbox workspace.
// Missing parent directories are created automatically; an existing file is
// overwritten silently.
func Write(ctx context.Context, w FileWriter, in WriteInput) (WriteOutput, error) {
	full, err := resolvePath(in.Path)
	if err != nil {
		return WriteOutput{}, err
	}

	if err := w.WriteFile(ctx, full, []byte(in.Content)); err != nil {
		return WriteOutput{}, fmt.Errorf("activities: write %q: %w", in.Path, err)
	}
	return WriteOutput{}, nil
}
