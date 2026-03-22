package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoopUsageReadsAgentopsCommand(t *testing.T) {
	repoRoot := t.TempDir()

	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example.com/test\n\ngo 1.25.5\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	cmdDir := filepath.Join(repoRoot, "cmd", "agentops")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatalf("mkdir cmd/agentops: %v", err)
	}

	const mainSource = `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "loop" {
		if len(os.Args) > 2 && os.Args[2] == "lab" {
			fmt.Println("command: loop lab")
			fmt.Println("status: fail")
			fmt.Println("failures:")
			fmt.Println("- [usage] usage: agentops loop lab [compare|replay|run|judge|autofix] [advanced flags]")
			os.Exit(2)
		}
		fmt.Println("command: loop")
		fmt.Println("status: fail")
		fmt.Println("failures:")
		fmt.Println("- [usage] usage: agentops loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]")
		os.Exit(2)
	}
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainSource), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	leanSig, err := loopUsage(repoRoot, "loop")
	if err != nil {
		t.Fatalf("loopUsage(loop): %v", err)
	}
	if !strings.Contains(leanSig, "agentops loop") {
		t.Fatalf("expected agentops loop signature, got: %s", leanSig)
	}

	labSig, err := loopUsage(repoRoot, "loop", "lab")
	if err != nil {
		t.Fatalf("loopUsage(loop, lab): %v", err)
	}
	if leanSig != "agentops loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]" {
		t.Fatalf("unexpected lean signature: %q", leanSig)
	}
	if labSig != "agentops loop lab [compare|replay|run|judge|autofix] [advanced flags]" {
		t.Fatalf("unexpected lab signature: %q", labSig)
	}
}
