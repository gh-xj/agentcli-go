package harnessloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildLocalAgentCLIBinaryBuildsAgentopsBinary(t *testing.T) {
	repoRoot := t.TempDir()

	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n\ngo 1.25.5\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmdDir := filepath.Join(repoRoot, "cmd", "agentops")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatalf("mkdir cmd/agentops: %v", err)
	}

	const mainSource = `package main

func main() {}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainSource), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	binPath, err := BuildLocalAgentCLIBinary(repoRoot)
	if err != nil {
		t.Fatalf("BuildLocalAgentCLIBinary: %v", err)
	}
	if filepath.Base(binPath) != "agentops-loop" {
		t.Fatalf("expected agentops-loop binary name, got %q", filepath.Base(binPath))
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("stat built binary: %v", err)
	}
}
