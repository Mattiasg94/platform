package pod

import "context"

type Result struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Diff    string `json:"diff"`
}

// Runner is the pod seam (ADR-0004): the isolation tech is swappable behind it.
type Runner interface {
	EnsureImage(ctx context.Context) error
	Run(ctx context.Context, prompt string) (Result, error)
	Verify(ctx context.Context) (Verification, error)
}
