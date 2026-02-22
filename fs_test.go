package agentcli

import (
	"path/filepath"
	"testing"
)

func TestEnsureDirAndFileExists(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "a", "b", "c")
	if err := EnsureDir(target); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}
	if !FileExists(target) {
		t.Fatalf("expected path to exist: %s", target)
	}
}

func TestGetBaseName(t *testing.T) {
	if got := GetBaseName("/tmp/archive.tar.gz"); got != "archive.tar" {
		t.Fatalf("basename mismatch: %q", got)
	}
}
