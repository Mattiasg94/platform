// Package activities holds the orchestrator's typed coding capabilities
// (Read, Write, Exec). Each function is a standalone, provider-agnostic
// "Activity": typed input in, typed output out, no awareness of any model
// provider's tool-call format. This split exists so they can later be
// registered as Temporal Activities with little to no rewriting.
package activities

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"orchestrator/internal/sandbox"
)

// WorkspaceRoot is the fixed root all tool paths resolve against.
const WorkspaceRoot = sandbox.WorkspaceRoot

// ErrPathTraversal is returned when a tool path attempts to escape the
// workspace root.
var ErrPathTraversal = errors.New("activities: path escapes workspace root")

// resolvePath joins p onto the workspace root, treating p as relative
// regardless of a leading slash, and rejects any attempt to traverse above
// the root (e.g. "../secret").
func resolvePath(p string) (string, error) {
	trimmed := strings.TrimPrefix(p, "/")
	cleaned := path.Clean(trimmed)
	if cleaned == "." {
		return WorkspaceRoot, nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: %q", ErrPathTraversal, p)
	}
	return path.Join(WorkspaceRoot, cleaned), nil
}
