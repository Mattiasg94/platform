package sandbox

import "context"

// Sandbox is the isolation seam (ADR-0004): a container the orchestrator can
// create and tear down. Kept deliberately small — it grows only when a slice
// needs it to.
type Sandbox interface {
	Start(ctx context.Context) error
	Destroy(ctx context.Context) error
}
