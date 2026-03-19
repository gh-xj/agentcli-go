package service

import (
	"path/filepath"
	"slices"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

const rootCommandMarker = "// agentcli:add-command"

// DoctorService checks project compliance against the golden scaffold contract.
type DoctorService struct {
	comp operator.ComplianceOperator
	fs   dal.FileSystem
}

// NewDoctorService returns a new DoctorService.
func NewDoctorService(comp operator.ComplianceOperator, fs dal.FileSystem) *DoctorService {
	return &DoctorService{comp: comp, fs: fs}
}

// Run performs all compliance checks using auto-detected mode.
func (s *DoctorService) Run(rootDir string) agentcli.DoctorReport {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	// Auto-detect: if internal/app exists, assume full mode
	mode := "lean"
	if s.fs.Exists(filepath.Join(rootDir, "internal", "app")) {
		mode = "full"
	}
	return s.RunWithMode(rootDir, mode)
}

// RunWithMode performs compliance checks scoped to the given mode.
// Mode "full" checks everything. Mode "lean" skips DAG and lifecycle checks.
func (s *DoctorService) RunWithMode(rootDir, mode string) agentcli.DoctorReport {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}

	report := agentcli.DoctorReport{
		SchemaVersion: "v1",
		OK:            true,
		Findings:      make([]agentcli.DoctorFinding, 0),
	}

	// Core files required in all modes
	required := []string{
		"main.go",
		"cmd/root.go",
		"internal/io/output.go",
		"internal/tools/smokecheck/main.go",
		"test/e2e/cli_test.go",
		"test/smoke/version.schema.json",
		"Taskfile.yml",
	}

	// Full mode adds lifecycle, config, version
	if mode == "full" {
		required = append(required,
			"internal/app/bootstrap.go",
			"internal/app/lifecycle.go",
			"internal/app/errors.go",
			"internal/config/schema.go",
			"internal/config/load.go",
			"pkg/version/version.go",
		)
	}

	// Check go.mod presence or parent module
	if s.fs.Exists(strings.TrimSpace(rootDir) + "/go.mod") {
		required = append(required, "go.mod")
	}

	// Required file checks
	for _, p := range required {
		if f := s.comp.CheckFileExists(rootDir, p); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}

	// Content checks (core - all modes)
	contentChecks := []struct {
		path string
		code string
		want string
		msg  string
	}{
		{"cmd/root.go", "missing_contract", `"github.com/gh-xj/agentcli-go/cobrax"`, "cobrax runtime import missing"},
		{"cmd/root.go", "missing_contract", rootCommandMarker, "missing scaffold command marker"},
		{"Taskfile.yml", "missing_contract", "ci:", "canonical CI task missing"},
		{"Taskfile.yml", "missing_contract", "verify:", "local verification task missing"},
		{"Taskfile.yml", "missing_contract", "test/smoke/version.output.json", "smoke artifact output path missing"},
		{"Taskfile.yml", "missing_contract", "internal/tools/smokecheck", "smoke schema validation command missing"},
		{"test/smoke/version.schema.json", "missing_contract", `"schema_version": "v1"`, "smoke schema version missing"},
	}

	// Full-mode-only content checks
	if mode == "full" {
		contentChecks = append(contentChecks,
			struct {
				path string
				code string
				want string
				msg  string
			}{"internal/app/lifecycle.go", "missing_contract", "Preflight", "lifecycle preflight hook missing"},
			struct {
				path string
				code string
				want string
				msg  string
			}{"internal/app/lifecycle.go", "missing_contract", "Postflight", "lifecycle postflight hook missing"},
		)
	}

	for _, c := range contentChecks {
		if f := s.comp.CheckFileContains(rootDir, c.path, c.code, c.want, c.msg); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}

	// DAG compliance checks (full mode only)
	if mode == "full" {
		dagChecks := []struct {
			path string
			code string
			want string
			msg  string
		}{
			{"service/wire.go", "missing_contract", "ProviderSet", "Wire provider set missing in service/wire.go"},
			{"dal/interfaces.go", "missing_contract", "dal", "dal package declaration missing"},
			{"operator/interfaces.go", "missing_contract", "operator", "operator package declaration missing"},
		}

		for _, c := range dagChecks {
			if f := s.comp.CheckFileContains(rootDir, c.path, c.code, c.want, c.msg); f != nil {
				report.Findings = append(report.Findings, *f)
			}
		}
	}

	// Sort by path then code
	slices.SortFunc(report.Findings, func(a, b agentcli.DoctorFinding) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return strings.Compare(a.Code, b.Code)
	})

	report.OK = len(report.Findings) == 0
	return report
}
