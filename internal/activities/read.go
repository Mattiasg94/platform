package activities

import (
	"context"
	"fmt"
)

// FileReader is the sandbox capability Read depends on.
type FileReader interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
}

type ReadInput struct {
	Path string
}

type ReadOutput struct {
	Content string
}

// Read returns a file's contents from the sandbox workspace. A nonexistent
// file is a genuine error, not a result to hand back as data.
func Read(ctx context.Context, r FileReader, in ReadInput) (ReadOutput, error) {
	full, err := resolvePath(in.Path)
	if err != nil {
		return ReadOutput{}, err
	}

	content, err := r.ReadFile(ctx, full)
	if err != nil {
		return ReadOutput{}, fmt.Errorf("activities: read %q: %w", in.Path, err)
	}
	return ReadOutput{Content: string(content)}, nil
}
