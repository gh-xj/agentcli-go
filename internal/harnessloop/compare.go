package harnessloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CompareReport struct {
	SchemaVersion string       `json:"schema_version"`
	RunA          RunResult    `json:"run_a"`
	RunB          RunResult    `json:"run_b"`
	Delta         CompareDelta `json:"delta"`
}

type CompareDelta struct {
	Score             float64 `json:"score"`
	PassDelta         int     `json:"pass_delta"`
	FindingsDelta     int     `json:"findings_delta"`
	IterationsDelta   int     `json:"iterations_delta"`
	FixesAppliedDelta int     `json:"fixes_applied_delta"`
}

func CompareRuns(repoRoot, runA, runB string) (CompareReport, error) {
	a, err := loadRunByRef(repoRoot, runA)
	if err != nil {
		return CompareReport{}, err
	}
	b, err := loadRunByRef(repoRoot, runB)
	if err != nil {
		return CompareReport{}, err
	}
	passA, passB := 0, 0
	if a.Judge.Pass {
		passA = 1
	}
	if b.Judge.Pass {
		passB = 1
	}
	return CompareReport{
		SchemaVersion: "v1",
		RunA:          a,
		RunB:          b,
		Delta: CompareDelta{
			Score:             b.Judge.Score - a.Judge.Score,
			PassDelta:         passB - passA,
			FindingsDelta:     len(b.Findings) - len(a.Findings),
			IterationsDelta:   b.Iterations - a.Iterations,
			FixesAppliedDelta: len(b.FixesApplied) - len(a.FixesApplied),
		},
	}, nil
}

func loadRunByRef(repoRoot, runRef string) (RunResult, error) {
	path := strings.TrimSpace(runRef)
	if path == "" {
		return RunResult{}, fmt.Errorf("empty run reference")
	}
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		return loadRun(path)
	}
	candidate := filepath.Join(repoRoot, ".docs", "onboarding-loop", "runs", path, "final-report.json")
	if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
		return loadRun(candidate)
	}
	return RunResult{}, fmt.Errorf("run report not found: %s", runRef)
}

func loadRun(path string) (RunResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RunResult{}, err
	}
	var out RunResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return RunResult{}, fmt.Errorf("parse run report %s: %w", path, err)
	}
	return out, nil
}
