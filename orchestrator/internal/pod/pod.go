package pod

import "context"

// Result is what a finished agent pod hands back — the pod's I/O contract
// (ADR-0007): task in, structured result out. The orchestrator supervises the
// pod as one bounded job and consumes this; it never pipes individual tool
// calls.
type Result struct {
	Status  string `json:"status"`  // "success" | "error"
	Summary string `json:"summary"` // the harness's final message
	Diff    string `json:"diff"`    // git diff of the workspace it edited
}

// Runner launches the agent pod for one task and returns its result. Docker is
// today's implementation; the isolation tech is swappable behind this seam
// (ADR-0004) without touching callers.
type Runner interface {
	Run(ctx context.Context, prompt string) (Result, error)
}
