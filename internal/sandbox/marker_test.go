package sandbox

import "testing"

func TestNewMarker_ReturnsDifferentValuesEachCall(t *testing.T) {
	a, err := newMarker()
	if err != nil {
		t.Fatalf("newMarker: %v", err)
	}
	b, err := newMarker()
	if err != nil {
		t.Fatalf("newMarker: %v", err)
	}
	if a == b {
		t.Fatalf("newMarker() returned the same value twice: %q", a)
	}
}
