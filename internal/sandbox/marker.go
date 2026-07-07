package sandbox

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// newMarker returns a random, unique-per-call token used to bound one
// command's output in the persistent shell's stream. Random (not
// sequential) so a command's own output can't be crafted to collide with it.
func newMarker() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("sandbox: generate marker: %w", err)
	}
	return "__EXEC_MARKER_" + hex.EncodeToString(b), nil
}
