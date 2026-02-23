package harnessloop

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runPlannerRole(repoRoot, artifactDir string, spec RoleSpec, ctx roleContext) (plannerOutput, RoleExecution, error) {
	execMeta := RoleExecution{Strategy: strategyOrBuiltin(spec), Command: spec.Command, Artifacts: artifactDir}
	out := plannerOutput{SchemaVersion: "v1", Summary: "builtin planner", FixTargets: findingCodes(ctx.Findings)}

	if strings.TrimSpace(spec.Command) == "" {
		if err := validatePlannerOutput(out); err != nil {
			return out, execMeta, err
		}
		_ = writeJSON(filepath.Join(artifactDir, "planner-output.json"), out)
		return out, execMeta, nil
	}

	var external plannerOutput
	code, stderrTail, err := runExternalRole(spec.Command, repoRoot, artifactDir, "planner", ctx, &external)
	execMeta.ExitCode = code
	execMeta.StderrTail = stderrTail
	if err != nil {
		return out, execMeta, err
	}
	if external.SchemaVersion == "" {
		external.SchemaVersion = "v1"
	}
	if err := validatePlannerOutput(external); err != nil {
		return out, execMeta, err
	}
	_ = writeJSON(filepath.Join(artifactDir, "planner-output.json"), external)
	return external, execMeta, nil
}

func runFixerRole(repoRoot, artifactDir string, spec RoleSpec, ctx roleContext, findings []Finding, plan plannerOutput) ([]string, RoleExecution, error) {
	execMeta := RoleExecution{Strategy: strategyOrBuiltin(spec), Command: spec.Command, Artifacts: artifactDir}
	applied, err := ApplyFixes(repoRoot, findings)
	if err != nil {
		return nil, execMeta, err
	}
	fallback := fixerOutput{SchemaVersion: "v1", Applied: applied, Notes: "builtin fixer"}

	if strings.TrimSpace(spec.Command) == "" {
		if err := validateFixerOutput(fallback); err != nil {
			return nil, execMeta, err
		}
		execMeta.Applied = fallback.Applied
		execMeta.Notes = fallback.Notes
		_ = writeJSON(filepath.Join(artifactDir, "fixer-output.json"), fallback)
		return fallback.Applied, execMeta, nil
	}

	payload := struct {
		Context roleContext   `json:"context"`
		Plan    plannerOutput `json:"plan"`
	}{
		Context: ctx,
		Plan:    plan,
	}
	var external fixerOutput
	code, stderrTail, err := runExternalRole(spec.Command, repoRoot, artifactDir, "fixer", payload, &external)
	execMeta.ExitCode = code
	execMeta.StderrTail = stderrTail
	if err != nil {
		return fallback.Applied, execMeta, err
	}
	if external.SchemaVersion == "" {
		external.SchemaVersion = "v1"
	}
	if err := validateFixerOutput(external); err != nil {
		return fallback.Applied, execMeta, err
	}
	execMeta.Applied = external.Applied
	execMeta.Notes = external.Notes
	_ = writeJSON(filepath.Join(artifactDir, "fixer-output.json"), external)
	return append(fallback.Applied, external.Applied...), execMeta, nil
}

func runJudgerRole(repoRoot, artifactDir string, spec RoleSpec, ctx roleContext) (judgerOutput, RoleExecution, error) {
	execMeta := RoleExecution{Strategy: strategyOrBuiltin(spec), Command: spec.Command, Artifacts: artifactDir, Independent: true}
	fallback := judgerOutput{SchemaVersion: "v1", ExtraFindings: nil, Notes: "builtin judger"}
	if strings.TrimSpace(spec.Command) == "" {
		if err := validateJudgerOutput(fallback); err != nil {
			return fallback, execMeta, err
		}
		execMeta.Notes = fallback.Notes
		_ = writeJSON(filepath.Join(artifactDir, "judger-output.json"), fallback)
		return fallback, execMeta, nil
	}

	var external judgerOutput
	code, stderrTail, err := runExternalRole(spec.Command, repoRoot, artifactDir, "judger", ctx, &external)
	execMeta.ExitCode = code
	execMeta.StderrTail = stderrTail
	if err != nil {
		return fallback, execMeta, err
	}
	if external.SchemaVersion == "" {
		external.SchemaVersion = "v1"
	}
	if err := validateJudgerOutput(external); err != nil {
		return fallback, execMeta, err
	}
	execMeta.Notes = external.Notes
	_ = writeJSON(filepath.Join(artifactDir, "judger-output.json"), external)
	return external, execMeta, nil
}

func runExternalRole(command, repoRoot, artifactDir, role string, input any, out any) (int, string, error) {
	ctxPath := filepath.Join(artifactDir, role+"-context.json")
	outPath := filepath.Join(artifactDir, role+"-external-output.json")
	if err := writeJSON(ctxPath, input); err != nil {
		return 1, "", err
	}

	cmd := exec.Command("zsh", "-lc", command)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"HARNESS_ROLE="+role,
		"HARNESS_CONTEXT_FILE="+ctxPath,
		"HARNESS_OUTPUT_FILE="+outPath,
		"HARNESS_REPO_ROOT="+repoRoot,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			exitCode = 1
		}
	}
	stderrTail := tail(stderr.String(), 800)
	_ = os.WriteFile(filepath.Join(artifactDir, role+"-stdout.log"), stdout.Bytes(), 0644)
	_ = os.WriteFile(filepath.Join(artifactDir, role+"-stderr.log"), stderr.Bytes(), 0644)

	if err != nil {
		return exitCode, stderrTail, fmt.Errorf("external role %s failed: %w", role, err)
	}

	payload := bytes.TrimSpace(stdout.Bytes())
	if b, readErr := os.ReadFile(outPath); readErr == nil && len(bytes.TrimSpace(b)) > 0 {
		payload = b
	}
	if len(payload) == 0 {
		return exitCode, stderrTail, fmt.Errorf("external role %s produced no output", role)
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return exitCode, stderrTail, fmt.Errorf("parse external role %s output: %w", role, err)
	}
	return exitCode, stderrTail, nil
}

func findingCodes(findings []Finding) []string {
	codes := make([]string, 0, len(findings))
	for _, f := range findings {
		codes = append(codes, f.Code)
	}
	return codes
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func validatePlannerOutput(out plannerOutput) error {
	if out.SchemaVersion != "v1" {
		return fmt.Errorf("invalid planner schema_version: %q", out.SchemaVersion)
	}
	if strings.TrimSpace(out.Summary) == "" {
		return fmt.Errorf("planner summary is required")
	}
	return nil
}

func validateFixerOutput(out fixerOutput) error {
	if out.SchemaVersion != "v1" {
		return fmt.Errorf("invalid fixer schema_version: %q", out.SchemaVersion)
	}
	if strings.TrimSpace(out.Notes) == "" {
		return fmt.Errorf("fixer notes are required")
	}
	return nil
}

func validateJudgerOutput(out judgerOutput) error {
	if out.SchemaVersion != "v1" {
		return fmt.Errorf("invalid judger schema_version: %q", out.SchemaVersion)
	}
	if strings.TrimSpace(out.Notes) == "" {
		return fmt.Errorf("judger notes are required")
	}
	return nil
}
