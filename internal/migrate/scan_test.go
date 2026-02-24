package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanScriptsDetectsBashAndSh(t *testing.T) {
	repoRoot := t.TempDir()
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}

	bashScript := "#!/usr/bin/env bash\nset -euo pipefail\necho \"$1\"\necho \"$HOME\"\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "build.sh"), []byte(bashScript), 0o755); err != nil {
		t.Fatalf("write bash script: %v", err)
	}

	shScript := "#!/bin/sh\necho \"$1\"\n"
	if err := os.WriteFile(filepath.Join(scriptsDir, "cleanup"), []byte(shScript), 0o755); err != nil {
		t.Fatalf("write sh script: %v", err)
	}

	if err := os.WriteFile(filepath.Join(scriptsDir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	result, err := ScanScripts(repoRoot, "scripts")
	if err != nil {
		t.Fatalf("scan scripts: %v", err)
	}
	if len(result.Scripts) != 2 {
		t.Fatalf("unexpected script count: got %d want %d", len(result.Scripts), 2)
	}

	scriptByName := map[string]ScriptInfo{}
	for _, script := range result.Scripts {
		scriptByName[script.Name] = script
	}

	build, ok := scriptByName["build"]
	if !ok {
		t.Fatalf("expected build script in result: %+v", result.Scripts)
	}
	if build.Shell != ShellBash {
		t.Fatalf("expected bash shell, got %s", build.Shell)
	}
	if !build.UsesArgs {
		t.Fatalf("expected build script to use args")
	}
	if !build.UsesEnv {
		t.Fatalf("expected build script to use env")
	}

	cleanup, ok := scriptByName["cleanup"]
	if !ok {
		t.Fatalf("expected cleanup script in result: %+v", result.Scripts)
	}
	if cleanup.Shell != ShellSh {
		t.Fatalf("expected sh shell, got %s", cleanup.Shell)
	}
}
