package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/api/types/mount"
)

func TestDemoProjectHostPath_PointsAtDemoProjectDir(t *testing.T) {
	got := demoProjectHostPath()

	if filepath.Base(got) != "demo-project" {
		t.Fatalf("demoProjectHostPath() = %q, want a path ending in demo-project", got)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("demoProjectHostPath() = %q, want an absolute path", got)
	}
	if _, err := os.Stat(filepath.Join(got, "go.mod")); err != nil {
		t.Fatalf("demoProjectHostPath() = %q does not contain go.mod: %v", got, err)
	}
}

func TestBuildMounts_BindsDemoProjectAtWorkspaceRoot(t *testing.T) {
	mounts := buildMounts()

	if len(mounts) != 1 {
		t.Fatalf("buildMounts() returned %d mounts, want 1", len(mounts))
	}
	m := mounts[0]
	if m.Type != mount.TypeBind {
		t.Fatalf("mount.Type = %q, want bind", m.Type)
	}
	if m.Target != WorkspaceRoot {
		t.Fatalf("mount.Target = %q, want %q", m.Target, WorkspaceRoot)
	}
	if m.Source != demoProjectHostPath() {
		t.Fatalf("mount.Source = %q, want %q", m.Source, demoProjectHostPath())
	}
}
