package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
)

func TestRunAddCommandWithDescription(t *testing.T) {
	root := t.TempDir()
	projectPath, err := agentcli.ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	exitCode := run([]string{
		"add",
		"command",
		"--dir", projectPath,
		"--description", "sync files from source to target",
		"sync-data",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read generated command file: %v", err)
	}
	if !strings.Contains(string(content), `Description: "sync files from source to target"`) {
		t.Fatalf("expected description in generated command file: %s", string(content))
	}
}

func TestRunAddCommandDescriptionRequiresValue(t *testing.T) {
	exitCode := run([]string{"add", "command", "--description"})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestRunAddCommandWithPreset(t *testing.T) {
	root := t.TempDir()
	projectPath, err := agentcli.ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	exitCode := run([]string{
		"add",
		"command",
		"--dir", projectPath,
		"--preset", "file-sync",
		"sync-data",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read generated command file: %v", err)
	}
	if !strings.Contains(string(content), `Description: "sync files between source and destination"`) {
		t.Fatalf("expected preset description in generated command file: %s", string(content))
	}
}

func TestRunAddCommandPresetRequiresValue(t *testing.T) {
	exitCode := run([]string{"add", "command", "--preset"})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestRunAddCommandListPresets(t *testing.T) {
	exitCode := run([]string{"add", "command", "--list-presets"})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}
}

func TestRunAddCommandRejectsUnknownPreset(t *testing.T) {
	root := t.TempDir()
	projectPath, err := agentcli.ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	exitCode := run([]string{
		"add",
		"command",
		"--dir", projectPath,
		"--preset", "unknown",
		"sync-data",
	})
	if exitCode != agentcli.ExitFailure {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitFailure)
	}
}

func TestRunAddCommandUsesPresetSpecificStub(t *testing.T) {
	root := t.TempDir()
	projectPath, err := agentcli.ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	exitCode := run([]string{
		"add",
		"command",
		"--dir", projectPath,
		"--preset", "http-client",
		"sync-data",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read generated command file: %v", err)
	}
	if !strings.Contains(string(content), `preset := "http-client"`) {
		t.Fatalf("expected preset marker in generated command file: %s", string(content))
	}
	if !strings.Contains(string(content), "preset=http-client: request plan ready") {
		t.Fatalf("expected preset-specific message in generated command file: %s", string(content))
	}
}

func TestRunLoopUnknownAction(t *testing.T) {
	exitCode := run([]string{"loop", "unknown"})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestParseLoopFlags(t *testing.T) {
	opts, err := parseLoopFlags([]string{
		"--repo-root", ".",
		"--threshold", "8.5",
		"--max-iterations", "2",
		"--branch", "autofix/test",
		"--api", "http://127.0.0.1:7878",
		"--mode", "committee",
		"--role-config", ".docs/roles.json",
		"--seed", "7",
		"--budget", "3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.RepoRoot != "." || opts.Threshold != 8.5 || opts.MaxIterations != 2 || opts.Branch != "autofix/test" || opts.APIURL != "http://127.0.0.1:7878" || opts.Mode != "committee" || opts.RoleConfig != ".docs/roles.json" || opts.Seed != 7 || opts.Budget != 3 {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopFlagsInvalidMode(t *testing.T) {
	_, err := parseLoopFlags([]string{"--mode", "random"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}
