package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	harness "github.com/gh-xj/agentcli-go/tools/harness"
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
	exitCode := run([]string{"loop", "capabilities", "--format", "json"})
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
	repoRoot, err := parseLoopProfilesRepoRoot([]string{"--repo-root", "/tmp/project"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoRoot != "/tmp/project" {
		t.Fatalf("expected repo root override, got %s", repoRoot)
	}
}

func TestParseLoopProfilesRepoRootMissingValue(t *testing.T) {
	_, err := parseLoopProfilesRepoRoot([]string{"--repo-root"})
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

	profiles, err := getLoopProfiles(root)
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}

	quality := profiles["quality"]
	if quality.mode != "single" || quality.roleConfig != "configs/custom.roles.json" || quality.maxIterations != 2 || quality.threshold != 9.5 || quality.budget != 2 || quality.verboseArtifacts {
		t.Fatalf("unexpected overridden quality profile: %+v", quality)
	}

	quick := profiles["quick"]
	if quick.mode != "committee" || quick.roleConfig != "configs/quick.roles.json" || quick.maxIterations != 1 || quick.threshold != 7.2 {
		t.Fatalf("unexpected quick profile: %+v", quick)
	}
}

func TestGetLoopProfilesMissingFile(t *testing.T) {
	root := t.TempDir()
	profiles, err := getLoopProfiles(root)
	if err != nil {
		t.Fatalf("load builtin profiles: %v", err)
	}
	if _, ok := profiles["quality"]; !ok {
		t.Fatalf("expected builtin quality profile")
	}
}

func TestParseLoopFlags(t *testing.T) {
	opts, err := parseLoopFlags([]string{
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

func TestParseLoopLabFlags(t *testing.T) {
	opts, err := parseLoopLabFlags([]string{
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

func TestParseLoopLabFlagsRejectMarkdown(t *testing.T) {
	_, err := parseLoopLabFlags([]string{"--md"})
	if err == nil {
		t.Fatal("expected error for --md in lab flags")
	}
}

func TestParseLoopLabFlagsInvalidMode(t *testing.T) {
	_, err := parseLoopLabFlags([]string{"--mode", "random"})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestParseLoopQualityFlags(t *testing.T) {
	opts, err := parseLoopQualityFlags(loopProfiles["quality"], []string{
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

func TestParseLoopQualityFlagsNoVerboseArtifacts(t *testing.T) {
	opts, err := parseLoopQualityFlags(loopProfiles["quality"], []string{
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
	_, err := parseLoopQualityFlags(loopProfiles["quality"], []string{
		"--verbose-artifacts",
		"--no-verbose-artifacts",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParseLoopLabFlagsNoVerboseArtifacts(t *testing.T) {
	opts, err := parseLoopLabFlags([]string{
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
	_, err := parseLoopLabFlags([]string{
		"--verbose-artifacts",
		"--no-verbose-artifacts",
	})
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestResolveVerboseArtifacts(t *testing.T) {
	got, err := resolveVerboseArtifacts(true, false, false)
	if err != nil || !got {
		t.Fatalf("expected default true, got=%v err=%v", got, err)
	}
	got, err = resolveVerboseArtifacts(true, false, true)
	if err != nil || got {
		t.Fatalf("expected forced false, got=%v err=%v", got, err)
	}
	got, err = resolveVerboseArtifacts(false, true, false)
	if err != nil || !got {
		t.Fatalf("expected forced true, got=%v err=%v", got, err)
	}
	_, err = resolveVerboseArtifacts(false, true, true)
	if err == nil {
		t.Fatal("expected conflict error")
	}
}

func TestParseLoopRegressionFlags(t *testing.T) {
	opts, remaining, err := parseLoopRegressionFlags([]string{
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
	_, _, err := parseLoopRegressionFlags([]string{"--profile"})
	if err == nil {
		t.Fatal("expected missing profile value error")
	}
	_, _, err = parseLoopRegressionFlags([]string{"--baseline"})
	if err == nil {
		t.Fatal("expected missing baseline value error")
	}
}

func TestResolveLoopRegressionBaselinePath(t *testing.T) {
	got := resolveLoopRegressionBaselinePath("/tmp/repo", "quality", "")
	want := filepath.Join("/tmp/repo", "testdata", "regression", "loop-quality.behavior-baseline.json")
	if got != want {
		t.Fatalf("unexpected default baseline path: got %s want %s", got, want)
	}
	custom := resolveLoopRegressionBaselinePath("/tmp/repo", "quality", "artifacts/baseline.json")
	if custom != filepath.Join("/tmp/repo", "artifacts", "baseline.json") {
		t.Fatalf("unexpected custom baseline path: %s", custom)
	}
	abs := resolveLoopRegressionBaselinePath("/tmp/repo", "quality", "/var/tmp/b.json")
	if abs != "/var/tmp/b.json" {
		t.Fatalf("unexpected absolute baseline path: %s", abs)
	}
}
