package harnessloop

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func BuildLocalAgentCLIBinary(repoRoot string) (string, error) {
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(absRoot, ".docs", "onboarding-loop", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	binPath := filepath.Join(binDir, "agentops-loop")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/agentops")
	cmd.Dir = absRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build agentops binary: %w\n%s", err, string(out))
	}
	return binPath, nil
}

func RunScenario(s Scenario) (ScenarioResult, error) {
	workDir := s.WorkDir
	if strings.TrimSpace(workDir) == "" {
		tmp, err := os.MkdirTemp("", "agentops-loop-")
		if err != nil {
			return ScenarioResult{}, err
		}
		workDir = tmp
	}

	result := ScenarioResult{
		Name:      s.Name,
		StartedAt: time.Now().UTC(),
		OK:        true,
		Steps:     make([]StepResult, 0, len(s.Steps)),
	}

	for _, step := range s.Steps {
		sr := runStep(workDir, step)
		result.Steps = append(result.Steps, sr)
		if sr.ExitCode != 0 {
			result.OK = false
			break
		}
	}

	result.FinishedAt = time.Now().UTC()
	return result, nil
}

func runStep(workDir string, step Step) StepResult {
	start := time.Now()
	cmd := exec.Command("zsh", "-lc", step.Command)
	cmd.Dir = workDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	duration := time.Since(start)
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = 1
		}
	}
	combined := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if len(combined) > 800 {
		combined = combined[len(combined)-800:]
	}

	return StepResult{
		Name:         step.Name,
		Command:      step.Command,
		ExitCode:     code,
		DurationMs:   duration.Milliseconds(),
		Stdout:       stdout.String(),
		Stderr:       stderr.String(),
		CombinedTail: combined,
	}
}
