package strategy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentops/strategy"
)

func TestDiscoverFromSubdir(t *testing.T) {
	tmp := t.TempDir()
	// Create .agentops/ at root
	agentopsDir := filepath.Join(tmp, ".agentops")
	if err := os.MkdirAll(agentopsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentopsDir, "storage.yaml"), []byte("backend: in-repo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentopsDir, "transitions.yaml"), []byte("categories:\n  active: [open]\ninitial: open\ntransitions: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory
	subdir := filepath.Join(tmp, "src", "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Load from subdirectory should find .agentops/ above
	strat, err := strategy.Discover(subdir)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if strat.Root != tmp {
		t.Errorf("Root = %q, want %q", strat.Root, tmp)
	}
	if strat.Storage.Backend != "in-repo" {
		t.Errorf("Backend = %q, want %q", strat.Storage.Backend, "in-repo")
	}
}

func TestDiscoverMissing(t *testing.T) {
	tmp := t.TempDir()
	_, err := strategy.Discover(tmp)
	if err == nil {
		t.Fatal("expected error when .agentops/ not found")
	}
}

func TestBootstrapCreatesDefaults(t *testing.T) {
	tmp := t.TempDir()
	err := strategy.Bootstrap(tmp)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}

	// Verify key files exist
	for _, name := range []string{"strategy.md", "schema.md", "slot.md", "slot.yaml", "storage.yaml", "transitions.yaml", "risk.yaml", "routing.yaml", "budget.yaml", "hooks.yaml"} {
		path := filepath.Join(tmp, ".agentops", name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected %s to exist", name)
		}
	}
}

func TestBootstrapIdempotent(t *testing.T) {
	tmp := t.TempDir()

	// First bootstrap
	if err := strategy.Bootstrap(tmp); err != nil {
		t.Fatalf("first Bootstrap failed: %v", err)
	}

	// Overwrite a file with custom content
	customPath := filepath.Join(tmp, ".agentops", "storage.yaml")
	custom := []byte("backend: in-repo\ncustom: true\n")
	if err := os.WriteFile(customPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	// Second bootstrap should not overwrite
	if err := strategy.Bootstrap(tmp); err != nil {
		t.Fatalf("second Bootstrap failed: %v", err)
	}

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(custom) {
		t.Errorf("Bootstrap overwrote existing file: got %q, want %q", string(data), string(custom))
	}
}
