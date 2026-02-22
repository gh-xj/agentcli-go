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

func TestScaffoldAddCommandWiresRoot(t *testing.T) {
	root := t.TempDir()
	projectPath, err := ScaffoldNew(root, "samplecli", "example.com/samplecli")
	if err != nil {
		t.Fatalf("ScaffoldNew failed: %v", err)
	}
	if err := ScaffoldAddCommand(projectPath, "sync-data"); err != nil {
		t.Fatalf("ScaffoldAddCommand failed: %v", err)
	}

	cmdBody, err := os.ReadFile(filepath.Join(projectPath, "cmd", "sync-data.go"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	if !strings.Contains(string(cmdBody), "func SyncDataCommand() command") {
		t.Fatalf("unexpected command function name: %s", string(cmdBody))
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
	if !strings.Contains(text, "require github.com/gh-xj/agentcli-go v0.2.0") {
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
	if err := ScaffoldAddCommand(projectPath, "sync-data"); err != nil {
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
