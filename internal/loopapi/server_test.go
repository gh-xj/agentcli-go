package loopapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLoopRoleConfig(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "configs", "roles.json")
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveLoopRoleConfig(absPath, root); err != nil || got != absPath {
		t.Fatalf("expected abs role config unchanged, got %s", got)
	}

	if got, err := resolveLoopRoleConfig("configs/roles.json", root); err != nil || got != absPath {
		t.Fatalf("expected relative role config to be resolved, got %s", got)
	}

	if got, err := resolveLoopRoleConfig("", root); err != nil || got != "" {
		t.Fatalf("expected empty role config to stay empty, got %s", got)
	}

	if _, err := resolveLoopRoleConfig("../../etc/passwd", root); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestResolveLoopRepoRoot(t *testing.T) {
	baseRoot := t.TempDir()
	child := filepath.Join(baseRoot, "repo", "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveLoopRepoRoot("", baseRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != baseRoot {
		t.Fatalf("expected default repo root, got %s", got)
	}
	got, err = resolveLoopRepoRoot("  "+child+"  ", baseRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != child {
		t.Fatalf("expected cleaned child repo root path, got %s", got)
	}
	outsideRoot := t.TempDir()
	if _, err := resolveLoopRepoRoot(outsideRoot, baseRoot); err == nil {
		t.Fatal("expected outside repo root to be rejected")
	}
}

func TestResolveLoopRoleConfigRejectsSymlinkEscapesRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.roles.json")
	if err := os.WriteFile(outside, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "escape.roles.json")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if _, err := resolveLoopRoleConfig("escape.roles.json", root); err == nil {
		t.Fatal("expected symlink traversal to be rejected")
	}
}

func TestResolveLoopRoleConfigRejectsMissingPathUnderSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "escape")
	if err := os.Symlink(outside, linkDir); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	if _, err := resolveLoopRoleConfig(filepath.Join("escape", "missing.roles.json"), root); err == nil {
		t.Fatal("expected symlink escape with missing target to be rejected")
	}
}
