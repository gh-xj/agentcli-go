package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	harness "github.com/gh-xj/agentcli-go/tools/harness"
	loopcommands "github.com/gh-xj/agentcli-go/tools/harness/commands"
)

func captureOutputs(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	doneOut := make(chan string, 1)
	doneErr := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = b.ReadFrom(rOut)
		doneOut <- b.String()
	}()
	go func() {
		var b bytes.Buffer
		_, _ = b.ReadFrom(rErr)
		doneErr <- b.String()
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	return <-doneOut, <-doneErr
}

func withWorkingDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	}()
	fn()
}

func TestRunHelpIncludesMigrateEntryAndAgentPrompt(t *testing.T) {
	_, stderr := captureOutputs(t, func() {
		exitCode := run([]string{"--help"})
		if exitCode != agentcli.ExitSuccess {
			t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
		}
	})
	if !strings.Contains(stderr, "agentcli migrate --source path [--mode safe|in-place] [--dry-run|--apply]") {
		t.Fatalf("expected migrate help usage in stderr: %s", stderr)
	}
	if !strings.Contains(stderr, "agent prompt: run 'agentcli migrate --source ./scripts --mode safe --dry-run' first") {
		t.Fatalf("expected agent prompt in help text: %s", stderr)
	}
}

func TestRunMigrateDryRunPrintsPlanWithoutWriting(t *testing.T) {
	repoRoot := t.TempDir()
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "sync.sh"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	withWorkingDir(t, repoRoot, func() {
		stdout, _ := captureOutputs(t, func() {
			exitCode := run([]string{"migrate", "--source", "scripts", "--mode", "safe", "--dry-run"})
			if exitCode != agentcli.ExitSuccess {
				t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
			}
		})
		if !strings.Contains(stdout, "migration plan (dry-run)") {
			t.Fatalf("expected dry-run summary in stdout: %s", stdout)
		}
	})

	if agentcli.FileExists(filepath.Join(repoRoot, "agentcli-migrated")) {
		t.Fatalf("dry-run should not create output workspace")
	}
}

func TestRunMigrateApplyCreatesSafeWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "sync.sh"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	withWorkingDir(t, repoRoot, func() {
		stdout, _ := captureOutputs(t, func() {
			exitCode := run([]string{"migrate", "--source", "scripts", "--mode", "safe", "--apply"})
			if exitCode != agentcli.ExitSuccess {
				t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
			}
		})
		if !strings.Contains(stdout, "migration generated at:") {
			t.Fatalf("expected apply output in stdout: %s", stdout)
		}
	})

	if !agentcli.FileExists(filepath.Join(repoRoot, "agentcli-migrated", "docs", "migration", "plan.json")) {
		t.Fatalf("expected migration plan artifact in safe workspace")
	}
}

func TestRunMigrateHelp(t *testing.T) {
	_, stderr := captureOutputs(t, func() {
		exitCode := run([]string{"migrate", "--help"})
		if exitCode != agentcli.ExitSuccess {
			t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
		}
	})
	if !strings.Contains(stderr, "usage: agentcli migrate --source path [--mode safe|in-place] [--dry-run|--apply] [--out path]") {
		t.Fatalf("expected migrate usage help: %s", stderr)
	}
}

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

func TestRunAddCommandTaskReplayOrchestratorPreset(t *testing.T) {
	root := t.TempDir()
	projectPath, err := agentcli.ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	exitCode := run([]string{
		"add",
		"command",
		"--dir", projectPath,
		"--preset", "task-replay-orchestrator",
		"replay-orchestrate",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, "cmd", "replay-orchestrate.go"))
	if err != nil {
		t.Fatalf("read generated command file: %v", err)
	}
	if !strings.Contains(string(content), "task-replay-orchestrator") {
		t.Fatalf("expected preset marker in generated command file: %s", string(content))
	}
	if !strings.Contains(string(content), "--timeout") || !strings.Contains(string(content), "--timeout-hook") {
		t.Fatalf("expected timeout hooks in generated command file: %s", string(content))
	}
}

func TestRunNewInExistingModuleMode(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/mono\n\ngo 1.25.5\n"), 0o644); err != nil {
		t.Fatalf("write module go.mod: %v", err)
	}

	exitCode := run([]string{
		"new",
		"--dir", filepath.Join(moduleRoot, "tools"),
		"--in-existing-module",
		"samplecli",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	projectPath := filepath.Join(moduleRoot, "tools", "samplecli")
	if agentcli.FileExists(filepath.Join(projectPath, "go.mod")) {
		t.Fatalf("expected no nested go.mod in existing-module mode")
	}
}

func TestRunNewRejectsModuleWithInExistingModuleMode(t *testing.T) {
	exitCode := run([]string{
		"new",
		"--in-existing-module",
		"--module", "example.com/custom",
		"samplecli",
	})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestRunNewMinimalMode(t *testing.T) {
	root := t.TempDir()
	exitCode := run([]string{
		"new",
		"--dir", root,
		"--minimal",
		"samplecli",
	})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}

	projectPath := filepath.Join(root, "samplecli")
	if !agentcli.FileExists(filepath.Join(projectPath, "go.mod")) || !agentcli.FileExists(filepath.Join(projectPath, "go.sum")) {
		t.Fatalf("expected go.mod and go.sum in minimal mode")
	}
	if agentcli.FileExists(filepath.Join(projectPath, "internal", "app", "bootstrap.go")) {
		t.Fatalf("did not expect full scaffold internals in minimal mode")
	}
}

func TestRunLoopUnknownAction(t *testing.T) {
	exitCode := run([]string{"loop", "unknown"})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestRunLoopProfileRequiresName(t *testing.T) {
	exitCode := run([]string{"loop", "profile"})
	if exitCode != agentcli.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitUsage)
	}
}

func TestRunLoopNamedProfileAlias(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "configs", "loop-profiles.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	raw := `{
  "lean": {
    "mode": "committee",
    "role_config": "configs/committee.roles.example.json",
    "max_iterations": 1,
    "threshold": 7.5,
    "budget": 1,
    "verbose_artifacts": false
  }
}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write loop profiles: %v", err)
	}

	exitCode := run([]string{"loop", "lean", "--repo-root", root, "--api", "http://127.0.0.1:0"})
	if exitCode != harness.ExitExecutionFailure {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, harness.ExitExecutionFailure)
	}
}

func TestRunLoopProfileSubcommand(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "configs", "loop-profiles.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	raw := `{
  "lean": {
    "mode": "committee",
    "role_config": "configs/committee.roles.example.json",
    "max_iterations": 1,
    "threshold": 7.5,
    "budget": 1,
    "verbose_artifacts": false
  }
}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write loop profiles: %v", err)
	}

	exitCode := run([]string{"loop", "profile", "lean", "--repo-root", root, "--api", "http://127.0.0.1:0"})
	if exitCode != harness.ExitExecutionFailure {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, harness.ExitExecutionFailure)
	}
}

func TestRunLoopCapabilities(t *testing.T) {
	exitCode := run([]string{"loop", "--format", "json", "capabilities"})
	if exitCode != harness.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, harness.ExitSuccess)
	}
}

func TestRunLoopCapabilitiesRejectsTrailingRuntimeFlags(t *testing.T) {
	exitCode := run([]string{"loop", "capabilities", "--format", "json"})
	if exitCode != harness.ExitUsage {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, harness.ExitUsage)
	}
}

func TestRunLoopDryRunSkipsExecution(t *testing.T) {
	exitCode := run([]string{
		"loop",
		"--dry-run",
		"--format", "json",
		"run",
		"--repo-root", ".",
		"--api", "http://127.0.0.1:0",
	})
	if exitCode != harness.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, harness.ExitSuccess)
	}
}

func TestRunLoopDoctor(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	exitCode := run([]string{"loop", "doctor", "--repo-root", repoRoot})
	if exitCode != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: got %d want %d", exitCode, agentcli.ExitSuccess)
	}
}

func TestParseLoopProfilesRepoRoot(t *testing.T) {
	repoRoot, err := loopcommands.ParseLoopProfilesRepoRoot([]string{"--repo-root", "/tmp/project"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoRoot != "/tmp/project" {
		t.Fatalf("expected repo root override, got %s", repoRoot)
	}
}

func TestParseLoopProfilesRepoRootMissingValue(t *testing.T) {
	_, err := loopcommands.ParseLoopProfilesRepoRoot([]string{"--repo-root"})
	if err == nil {
		t.Fatal("expected missing-value error")
	}
}

func TestGetLoopProfilesUsesBuiltinAndFile(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "configs", "loop-profiles.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir configs: %v", err)
	}
	raw := map[string]struct {
		Mode             string  `json:"mode"`
		RoleConfig       string  `json:"role_config"`
		MaxIterations    int     `json:"max_iterations"`
		Threshold        float64 `json:"threshold"`
		Budget           int     `json:"budget"`
		VerboseArtifacts bool    `json:"verbose_artifacts"`
	}{
		"quality": {
			Mode:             "single",
			RoleConfig:       "configs/custom.roles.json",
			MaxIterations:    2,
			Threshold:        9.5,
			Budget:           2,
			VerboseArtifacts: false,
		},
		"quick": {
			Mode:          "committee",
			RoleConfig:    "configs/quick.roles.json",
			MaxIterations: 1,
			Threshold:     7.2,
			Budget:        1,
		},
	}
	out, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal profile config: %v", err)
	}
	if err := os.WriteFile(configPath, out, 0o644); err != nil {
		t.Fatalf("write profile config: %v", err)
	}

	profiles, err := loopcommands.GetLoopProfiles(root)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}

	quality := profiles["quality"]
	if quality.Mode != "single" || quality.RoleConfig != "configs/custom.roles.json" || quality.MaxIterations != 2 || quality.Threshold != 9.5 || quality.Budget != 2 || quality.VerboseArtifacts {
		t.Fatalf("unexpected overridden quality profile: %+v", quality)
	}

	quick := profiles["quick"]
	if quick.Mode != "committee" || quick.RoleConfig != "configs/quick.roles.json" || quick.MaxIterations != 1 || quick.Threshold != 7.2 {
		t.Fatalf("unexpected quick profile: %+v", quick)
	}
}

func TestGetLoopProfilesMissingFile(t *testing.T) {
	root := t.TempDir()
	profiles, err := loopcommands.GetLoopProfiles(root)
	if err != nil {
		t.Fatalf("load builtin profiles: %v", err)
	}
	if _, ok := profiles["quality"]; !ok {
		t.Fatalf("expected builtin quality profile")
	}
}

func TestParseLoopFlags(t *testing.T) {
	opts, err := loopcommands.ParseLoopFlags([]string{
		"--repo-root", ".",
		"--threshold", "8.5",
		"--max-iterations", "2",
		"--branch", "autofix/test",
		"--api", "http://127.0.0.1:7878",
		"--md",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.RepoRoot != "." || opts.Threshold != 8.5 || opts.MaxIterations != 2 || opts.Branch != "autofix/test" || opts.APIURL != "http://127.0.0.1:7878" || !opts.Markdown {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopFlagsGlobalFlagPlacementHint(t *testing.T) {
	_, err := loopcommands.ParseLoopFlags([]string{"--format", "json"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "global flag; place it before the action") {
		t.Fatalf("expected placement hint, got: %v", err)
	}
}

func TestParseLoopLabFlags(t *testing.T) {
	opts, err := loopcommands.ParseLoopLabFlags([]string{
		"--repo-root", ".",
		"--threshold", "8.5",
		"--max-iterations", "2",
		"--branch", "autofix/test",
		"--api", "http://127.0.0.1:7878",
		"--mode", "committee",
		"--role-config", ".docs/roles.json",
		"--seed", "7",
		"--budget", "3",
		"--run-a", "runA",
		"--run-b", "runB",
		"--run-id", "runC",
		"--iter", "2",
		"--format", "md",
		"--out", ".docs/compare.md",
		"--verbose-artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.RepoRoot != "." || opts.Threshold != 8.5 || opts.MaxIterations != 2 || opts.Branch != "autofix/test" || opts.APIURL != "http://127.0.0.1:7878" || opts.Mode != "committee" || opts.RoleConfig != ".docs/roles.json" || opts.Seed != 7 || opts.Budget != 3 || opts.RunA != "runA" || opts.RunB != "runB" || opts.RunID != "runC" || opts.Iteration != 2 || opts.Format != "md" || opts.Out != ".docs/compare.md" || !opts.VerboseArtifacts {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopRuntimeFlagsLeadingGlobalsOnly(t *testing.T) {
	flags, remaining, err := loopcommands.ParseLoopRuntimeFlags([]string{
		"--format", "json",
		"--dry-run",
		"lab",
		"compare",
		"--format", "md",
		"--run-a", "runA",
		"--run-b", "runB",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags.Format != "json" || !flags.DryRun {
		t.Fatalf("unexpected runtime flags: %+v", flags)
	}
	if len(remaining) != 8 || remaining[0] != "lab" || remaining[2] != "--format" || remaining[3] != "md" {
		t.Fatalf("unexpected remaining args: %+v", remaining)
	}
}

func TestParseLoopRuntimeFlagsStopsAtAction(t *testing.T) {
	flags, remaining, err := loopcommands.ParseLoopRuntimeFlags([]string{
		"lab",
		"compare",
		"--format", "md",
		"--run-a", "runA",
		"--run-b", "runB",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flags.Format != "text" {
		t.Fatalf("unexpected runtime format: %s", flags.Format)
	}
	if len(remaining) != 8 || remaining[0] != "lab" || remaining[2] != "--format" || remaining[3] != "md" {
		t.Fatalf("unexpected remaining args: %+v", remaining)
	}
}

func TestParseLoopLabFlagsRejectMarkdown(t *testing.T) {
	_, err := loopcommands.ParseLoopLabFlags([]string{"--md"})
	if err == nil {
		t.Fatal("expected error for --md in lab flags")
	}
}

func TestParseLoopLabFlagsInvalidMode(t *testing.T) {
	_, err := loopcommands.ParseLoopLabFlags([]string{"--mode", "random"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestParseLoopQualityFlags(t *testing.T) {
	opts, err := loopcommands.ParseLoopQualityFlags(loopcommands.LoopProfiles["quality"], []string{
		"--repo-root", ".",
		"--threshold", "8.5",
		"--max-iterations", "2",
		"--branch", "autofix/test",
		"--api", "http://127.0.0.1:7878",
		"--role-config", ".docs/quality.roles.json",
		"--verbose-artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.RepoRoot != "." || opts.Threshold != 8.5 || opts.MaxIterations != 2 || opts.Branch != "autofix/test" || opts.APIURL != "http://127.0.0.1:7878" || opts.RoleConfig != ".docs/quality.roles.json" || !opts.VerboseArtifacts {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopQualityFlagsRejectMarkdown(t *testing.T) {
	_, err := loopcommands.ParseLoopQualityFlags(loopcommands.LoopProfiles["quality"], []string{"--md"})
	if err == nil {
		t.Fatal("expected error for --md in quality/profile flags")
	}
}

func TestParseLoopQualityFlagsNoVerboseArtifacts(t *testing.T) {
	opts, err := loopcommands.ParseLoopQualityFlags(loopcommands.LoopProfiles["quality"], []string{
		"--no-verbose-artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoVerboseArtifacts || opts.VerboseArtifacts {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopQualityFlagsVerboseConflict(t *testing.T) {
	_, err := loopcommands.ParseLoopQualityFlags(loopcommands.LoopProfiles["quality"], []string{
		"--verbose-artifacts",
		"--no-verbose-artifacts",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParseLoopLabFlagsNoVerboseArtifacts(t *testing.T) {
	opts, err := loopcommands.ParseLoopLabFlags([]string{
		"--no-verbose-artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.NoVerboseArtifacts || opts.VerboseArtifacts {
		t.Fatalf("unexpected parse values: %+v", opts)
	}
}

func TestParseLoopLabFlagsVerboseConflict(t *testing.T) {
	_, err := loopcommands.ParseLoopLabFlags([]string{
		"--verbose-artifacts",
		"--no-verbose-artifacts",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestResolveVerboseArtifacts(t *testing.T) {
	got, err := loopcommands.ResolveVerboseArtifacts(true, false, false)
	if err != nil || !got {
		t.Fatalf("expected default true, got=%v err=%v", got, err)
	}
	got, err = loopcommands.ResolveVerboseArtifacts(true, false, true)
	if err != nil || got {
		t.Fatalf("expected forced false, got=%v err=%v", got, err)
	}
	got, err = loopcommands.ResolveVerboseArtifacts(false, true, false)
	if err != nil || !got {
		t.Fatalf("expected forced true, got=%v err=%v", got, err)
	}
	_, err = loopcommands.ResolveVerboseArtifacts(false, true, true)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParseLoopRegressionFlags(t *testing.T) {
	opts, remaining, err := loopcommands.ParseLoopRegressionFlags([]string{
		"--profile", "lean",
		"--baseline", "testdata/regression/custom.json",
		"--write-baseline",
		"--repo-root", ".",
		"--no-verbose-artifacts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Profile != "lean" || opts.BaselinePath != "testdata/regression/custom.json" || !opts.WriteBaseline {
		t.Fatalf("unexpected regression flags: %+v", opts)
	}
	if len(remaining) != 3 || remaining[0] != "--repo-root" || remaining[1] != "." || remaining[2] != "--no-verbose-artifacts" {
		t.Fatalf("unexpected remaining args: %+v", remaining)
	}
}

func TestParseLoopRegressionFlagsMissingValue(t *testing.T) {
	_, _, err := loopcommands.ParseLoopRegressionFlags([]string{"--profile"})
	if err == nil {
		t.Fatal("expected missing profile value error")
	}
	_, _, err = loopcommands.ParseLoopRegressionFlags([]string{"--baseline"})
	if err == nil {
		t.Fatal("expected missing baseline value error")
	}
}

func TestResolveLoopRegressionBaselinePath(t *testing.T) {
	got := loopcommands.ResolveLoopRegressionBaselinePath("/tmp/repo", "quality", "")
	want := filepath.Join("/tmp/repo", "testdata", "regression", "loop-quality.behavior-baseline.json")
	if got != want {
		t.Fatalf("unexpected default baseline path: got %s want %s", got, want)
	}
	custom := loopcommands.ResolveLoopRegressionBaselinePath("/tmp/repo", "quality", "artifacts/baseline.json")
	if custom != filepath.Join("/tmp/repo", "artifacts", "baseline.json") {
		t.Fatalf("unexpected custom baseline path: %s", custom)
	}
	abs := loopcommands.ResolveLoopRegressionBaselinePath("/tmp/repo", "quality", "/var/tmp/b.json")
	if abs != "/var/tmp/b.json" {
		t.Fatalf("unexpected absolute baseline path: %s", abs)
	}
}
