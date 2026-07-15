package pod

import (
	"context"
	"fmt"
	"time"
)

// WorkspaceRoot is where the pod unpacks the tree it was given. The agent's own
// Dockerfile creates it; nothing is mounted there.
const WorkspaceRoot = "/workspace"

func newRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UTC().UnixMilli())
}

// Result is the pod's half of the I/O contract (ADR-0007). The agent writes this
// as JSON to the blackboard; nothing is scraped from its stdout.
type Result struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Diff    string `json:"diff"`
}

// store is the blackboard as this package needs it — the narrow slice, declared
// here so a fake can stand in and so the runner never learns what GCS is.
type store interface {
	PutTask(ctx context.Context, runID, task string) error
	PutWorkspace(ctx context.Context, runID, dir string) error
	GetResult(ctx context.Context, runID string) ([]byte, error)
}

// Runner is the pod seam (ADR-0004): the isolation tech is swappable behind it.
type Runner interface {
	Run(ctx context.Context, prompt string) (Result, error)
}
