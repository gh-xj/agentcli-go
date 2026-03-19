package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
)

// createLeanProject creates a temp dir with lean-mode files (no internal/app, no DAG).
func createLeanProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
		"cmd/root.go": "package cmd\n\nimport (\n\t\"github.com/gh-xj/agentcli-go/cobrax\"\n)\n\n" +
			"func init() {\n\t// agentcli:add-command\n}\n\nvar _ = cobrax.Execute\n",
		"internal/io/output.go":             "package appio\n",
		"internal/tools/smokecheck/main.go": "package main\n",
		"test/e2e/cli_test.go":              "package e2e\n",
		"test/smoke/version.schema.json":    `{"schema_version": "v1"}`,
		"Taskfile.yml":                      "ci:\nverify:\ntest/smoke/version.output.json\ninternal/tools/smokecheck\n",
		"go.mod":                            "module example\n\ngo 1.21\n",
	}

	for path, content := range files {
		abs := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
	}
	return dir
}

// createFullProject creates a temp dir with all required files passing doctor checks.
func createFullProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"main.go": "package main\n\nfunc main() {}\n",
		"cmd/root.go": "package cmd\n\nimport (\n\t\"github.com/gh-xj/agentcli-go/cobrax\"\n)\n\n" +
			"func init() {\n\t// agentcli:add-command\n}\n\nvar _ = cobrax.Execute\n",
		"internal/app/bootstrap.go":         "package app\n",
		"internal/app/lifecycle.go":         "package app\n\nfunc (Hooks) Preflight(_ *agentcli.AppContext) error { return nil }\nfunc (Hooks) Postflight(_ *agentcli.AppContext) error { return nil }\n",
		"internal/app/errors.go":            "package app\n",
		"internal/config/schema.go":         "package config\n",
		"internal/config/load.go":           "package config\n",
		"internal/io/output.go":             "package appio\n",
		"internal/tools/smokecheck/main.go": "package main\n",
		"pkg/version/version.go":            "package version\n",
		"test/e2e/cli_test.go":              "package e2e\n",
		"test/smoke/version.schema.json":    `{"schema_version": "v1"}`,
		"Taskfile.yml":                      "ci:\nverify:\ntest/smoke/version.output.json\ninternal/tools/smokecheck\n",
		"go.mod":                            "module example\n\ngo 1.21\n",
		"service/wire.go":                   "package service\n\nvar ProviderSet = 1\n",
		"dal/interfaces.go":                 "package dal\n",
		"operator/interfaces.go":            "package operator\n",
	}

	for path, content := range files {
		abs := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", abs, err)
		}
	}
	return dir
}

// osFS implements dal.FileSystem using real OS calls for testing.
type osFS struct{}

func (f *osFS) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *osFS) EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func (f *osFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *osFS) WriteFile(path string, data []byte, perm int) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

func (f *osFS) ReadDir(path string) ([]dal.DirEntry, error) {
	return nil, nil
}

func (f *osFS) BaseName(path string) string {
	return filepath.Base(path)
}

// osComplianceOp implements operator.ComplianceOperator using real file checks.
type osComplianceOp struct {
	fs dal.FileSystem
}

func (c *osComplianceOp) CheckFileExists(rootDir, relPath string) *agentcli.DoctorFinding {
	abs := filepath.Join(rootDir, relPath)
	if c.fs.Exists(abs) {
		return nil
	}
	return &agentcli.DoctorFinding{
		Code:    "missing_file",
		Path:    relPath,
		Message: "required file is missing",
	}
}

func (c *osComplianceOp) CheckFileContains(rootDir, relPath, code, want, msg string) *agentcli.DoctorFinding {
	abs := filepath.Join(rootDir, relPath)
	content, err := c.fs.ReadFile(abs)
	if err != nil {
		return nil
	}
	if !strings.Contains(string(content), want) {
		return &agentcli.DoctorFinding{
			Code:    code,
			Path:    relPath,
			Message: msg,
		}
	}
	return nil
}

func (c *osComplianceOp) ValidateCommandName(name string) error {
	return nil
}

func newDoctorSvc() *DoctorService {
	fs := &osFS{}
	comp := &osComplianceOp{fs: fs}
	return NewDoctorService(comp, fs)
}

func TestDoctorService_AllPassing(t *testing.T) {
	dir := createFullProject(t)
	svc := newDoctorSvc()

	report := svc.Run(dir)

	if !report.OK {
		t.Errorf("expected OK=true, got findings: %+v", report.Findings)
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %+v", len(report.Findings), report.Findings)
	}
}

func TestDoctorService_MissingFile(t *testing.T) {
	dir := createFullProject(t)
	os.Remove(filepath.Join(dir, "pkg/version/version.go"))

	svc := newDoctorSvc()
	report := svc.Run(dir)

	if report.OK {
		t.Error("expected OK=false when a required file is missing")
	}
	found := false
	for _, f := range report.Findings {
		if f.Path == "pkg/version/version.go" && f.Code == "missing_file" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding for pkg/version/version.go, got: %+v", report.Findings)
	}
}

func TestDoctorService_MissingContract(t *testing.T) {
	dir := createFullProject(t)
	os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("verify:\ntest/smoke/version.output.json\ninternal/tools/smokecheck\n"), 0644)

	svc := newDoctorSvc()
	report := svc.Run(dir)

	if report.OK {
		t.Error("expected OK=false when ci: is missing from Taskfile.yml")
	}
	found := false
	for _, f := range report.Findings {
		if f.Path == "Taskfile.yml" && f.Message == "canonical CI task missing" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding for missing ci: task, got: %+v", report.Findings)
	}
}

func TestDoctorService_DAGComplianceWireContent(t *testing.T) {
	dir := createFullProject(t)
	// Overwrite wire.go without ProviderSet
	os.WriteFile(filepath.Join(dir, "service/wire.go"), []byte("package service\n"), 0644)

	svc := newDoctorSvc()
	report := svc.Run(dir)

	if report.OK {
		t.Error("expected OK=false when ProviderSet is missing from service/wire.go")
	}
	found := false
	for _, f := range report.Findings {
		if f.Path == "service/wire.go" && strings.Contains(f.Message, "Wire provider set") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding for missing ProviderSet, got: %+v", report.Findings)
	}
}

func TestDoctorService_FindingsSorted(t *testing.T) {
	dir := createFullProject(t)
	os.Remove(filepath.Join(dir, "main.go"))
	os.Remove(filepath.Join(dir, "cmd/root.go"))

	svc := newDoctorSvc()
	report := svc.Run(dir)

	if report.OK {
		t.Error("expected OK=false")
	}

	for i := 1; i < len(report.Findings); i++ {
		prev := report.Findings[i-1]
		curr := report.Findings[i]
		if prev.Path > curr.Path || (prev.Path == curr.Path && prev.Code > curr.Code) {
			t.Errorf("findings not sorted: [%d](%s,%s) > [%d](%s,%s)", i-1, prev.Path, prev.Code, i, curr.Path, curr.Code)
		}
	}
}

func TestDoctorService_LeanModePassing(t *testing.T) {
	dir := createLeanProject(t)
	svc := newDoctorSvc()

	report := svc.RunWithMode(dir, "lean")

	if !report.OK {
		t.Errorf("expected OK=true for lean project, got findings: %+v", report.Findings)
	}
}

func TestDoctorService_LeanModeRejectsMissingCore(t *testing.T) {
	dir := createLeanProject(t)
	os.Remove(filepath.Join(dir, "cmd/root.go"))

	svc := newDoctorSvc()
	report := svc.RunWithMode(dir, "lean")

	if report.OK {
		t.Error("expected OK=false when core file missing in lean mode")
	}
}

func TestDoctorService_AutoDetectLeanMode(t *testing.T) {
	dir := createLeanProject(t)
	svc := newDoctorSvc()

	// Auto-detect: no internal/app/ dir means lean mode
	report := svc.Run(dir)

	if !report.OK {
		t.Errorf("expected auto-detect lean OK=true, got findings: %+v", report.Findings)
	}
}
