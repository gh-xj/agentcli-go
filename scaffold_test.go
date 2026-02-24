package agentcli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldNewCreatesGoldenLayout(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	required := []string{
		"go.mod",
		"main.go",
		"cmd/root.go",
		"internal/app/bootstrap.go",
		"internal/app/lifecycle.go",
		"internal/app/errors.go",
		"internal/config/schema.go",
		"internal/config/load.go",
		"internal/io/output.go",
		"internal/tools/smokecheck/main.go",
		"pkg/version/version.go",
		"test/e2e/cli_test.go",
		"test/smoke/version.schema.json",
		"Taskfile.yml",
	}
	for _, rel := range required {
		if !FileExists(filepath.Join(projectPath, rel)) {
			t.Fatalf("expected generated file: %s", rel)
		}
	}
}

func TestScaffoldNewGeneratesGoSum(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if !FileExists(filepath.Join(projectPath, "go.sum")) {
		t.Fatalf("expected generated file: go.sum")
	}
}

func TestScaffoldNewInExistingModuleSkipsGoModAndUsesParentImportPath(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/mono\n\ngo 1.25.5\n"), 0o644); err != nil {
		t.Fatalf("write module go.mod: %v", err)
	}

	baseDir := filepath.Join(moduleRoot, "tools")
	projectPath, err := ScaffoldNewWithOptions(baseDir, "replay-cli", "", ScaffoldNewOptions{
		InExistingModule: true,
	})
	if err != nil {
		t.Fatalf("ScaffoldNewWithOptions failed: %v", err)
	}
	if FileExists(filepath.Join(projectPath, "go.mod")) {
		t.Fatalf("expected no nested go.mod in existing-module mode")
	}

	mainBody, err := os.ReadFile(filepath.Join(projectPath, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainBody), `"example.com/mono/tools/replay-cli/cmd"`) {
		t.Fatalf("expected parent-module import path in main.go: %s", string(mainBody))
	}
}

func TestScaffoldNewInExistingModuleRequiresParentModule(t *testing.T) {
	root := t.TempDir()
	_, err := ScaffoldNewWithOptions(root, "samplecli", "", ScaffoldNewOptions{
		InExistingModule: true,
	})
	if err == nil {
		t.Fatal("expected error when parent module cannot be resolved")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("expected go.mod hint in error, got: %v", err)
	}
}

func TestScaffoldAddCommandWiresRoot(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "sync-data", "sync files from source to target", ""); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), "func SyncDataCommand() command") {
		t.Fatalf("unexpected command function name: %s", string(cmdBody))
	}
	if !strings.Contains(string(cmdBody), `Description: "sync files from source to target"`) {
		t.Fatalf("expected custom description in command body: %s", string(cmdBody))
	}

	rootBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "root.go"))
	if err != nil {
		t.Fatalf("read root.go: %v", err)
	}
	if !strings.Contains(string(rootBody), `registerCommand("sync-data", SyncDataCommand())`) {
		t.Fatalf("root command registration missing: %s", string(rootBody))
	}
}

func TestDoctorReportsGeneratedProjectAsOK(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	report := Doctor(projectPath)
	if !report.OK {
		t.Fatalf("expected doctor OK, findings: %+v", report.Findings)
	}
	if report.SchemaVersion != "v1" {
		t.Fatalf("unexpected schema version: %q", report.SchemaVersion)
	}
}

func TestDoctorAcceptsScaffoldInExistingModuleMode(t *testing.T) {
	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module example.com/mono\n\ngo 1.25.5\n"), 0o644); err != nil {
		t.Fatalf("write module go.mod: %v", err)
	}

	projectPath, err := ScaffoldNewWithOptions(filepath.Join(moduleRoot, "tools"), "samplecli", "", ScaffoldNewOptions{
		InExistingModule: true,
	})
	if err != nil {
		t.Fatalf("ScaffoldNewWithOptions failed: %v", err)
	}

	report := Doctor(projectPath)
	if !report.OK {
		t.Fatalf("expected doctor OK, findings: %+v", report.Findings)
	}
}

func TestDoctorDetectsMissingFile(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}

	missing := filepath.Join(projectPath, "internal", "config", "load.go")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	report := Doctor(projectPath)
	if report.OK {
		t.Fatal("expected doctor to fail")
	}
	found := false
	for _, f := range report.Findings {
		if f.Path == "internal/config/load.go" && f.Code == "missing_file" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing file finding not found: %+v", report.Findings)
	}
}

func TestDoctorJSONIncludesSchemaVersion(t *testing.T) {
	report := DoctorReport{
		SchemaVersion: "v1",
		OK:            true,
		Findings:      []DoctorFinding{},
	}
	out, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}
	if !strings.Contains(out, `"schema_version": "v1"`) {
		t.Fatalf("schema_version missing from JSON: %s", out)
	}
}

func TestScaffoldNewUsesCobraxRuntime(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	goMod, err := os.ReadFile(filepath.Join(projectPath, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	text := string(goMod)
	if !strings.Contains(text, "require github.com/gh-xj/agentcli-go v0.2.2") {
		t.Fatalf("missing phase2 requirement in go.mod: %s", text)
	}

	rootCmd, err := os.ReadFile(filepath.Join(projectPath, "cmd", "root.go"))
	if err != nil {
		t.Fatalf("read root.go: %v", err)
	}
	if !strings.Contains(string(rootCmd), `"github.com/gh-xj/agentcli-go/cobrax"`) {
		t.Fatalf("expected cobrax runtime import: %s", string(rootCmd))
	}
}

func TestScaffoldAddCommandUsesCobraxSignature(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "sync-data", "sync data command", ""); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), "func(app *agentcli.AppContext, args []string) error") {
		t.Fatalf("expected cobrax command signature, got: %s", string(cmdBody))
	}
}

func TestDoctorReportsCobraxProjectAsOK(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	report := Doctor(projectPath)
	if !report.OK {
		t.Fatalf("expected doctor OK for cobrax runtime, findings: %+v", report.Findings)
	}
}

func TestScaffoldAddCommandUsesDefaultDescriptionWhenMissing(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "sync-data", "", ""); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), `Description: "describe sync-data"`) {
		t.Fatalf("expected default description in command body: %s", string(cmdBody))
	}
}

func TestScaffoldAddCommandUsesPresetDescription(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "sync-data", "", "file-sync"); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), `Description: "sync files between source and destination"`) {
		t.Fatalf("expected preset description in command body: %s", string(cmdBody))
	}
	if !strings.Contains(string(cmdBody), `preset := "file-sync"`) {
		t.Fatalf("expected preset marker in command body: %s", string(cmdBody))
	}
	if !strings.Contains(string(cmdBody), "preset=file-sync: synced") {
		t.Fatalf("expected preset-specific output in command body: %s", string(cmdBody))
	}
}

func TestScaffoldAddCommandRejectsUnknownPreset(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	err = ScaffoldAddCommand(projectPath, "sync-data", "", "unknown")
	if err == nil {
		t.Fatal("expected error for unknown preset")
	}
	if !strings.Contains(err.Error(), "invalid preset") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCommandPresetNamesReturnsSortedNames(t *testing.T) {
	got := CommandPresetNames()
	want := []string{"deploy-helper", "file-sync", "http-client", "task-replay-emit-wrapper"}
	if len(got) != len(want) {
		t.Fatalf("unexpected length: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected preset at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestScaffoldAddCommandTaskReplayEmitWrapperPreset(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "replay-emit", "", "task-replay-emit-wrapper"); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "replay-emit.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), `Description: "run task replay emit wrapper in external repo with env injection"`) {
		t.Fatalf("expected preset description in command body: %s", string(cmdBody))
	}
	if !strings.Contains(string(cmdBody), "--repo") || !strings.Contains(string(cmdBody), "--env") {
		t.Fatalf("expected replay wrapper argument hints in command body: %s", string(cmdBody))
	}
}
