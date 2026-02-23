package harnessloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareRunsByID(t *testing.T) {
	repo := t.TempDir()
	runA := RunResult{SchemaVersion: "v1", RunID: "a", Judge: JudgeScore{Score: 8.5, Pass: false}, Findings: []Finding{{Code: "x"}}, Iterations: 2, FixesApplied: []string{"f1"}}
	runB := RunResult{SchemaVersion: "v1", RunID: "b", Judge: JudgeScore{Score: 9.5, Pass: true}, Findings: nil, Iterations: 1, FixesApplied: []string{"f1", "f2"}}

	pathA := filepath.Join(repo, ".docs", "onboarding-loop", "runs", "a", "final-report.json")
	pathB := filepath.Join(repo, ".docs", "onboarding-loop", "runs", "b", "final-report.json")
	if err := os.MkdirAll(filepath.Dir(pathA), 0755); err != nil {
		t.Fatalf("mkdir run A: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(pathB), 0755); err != nil {
		t.Fatalf("mkdir run B: %v", err)
	}
	if err := writeJSON(pathA, runA); err != nil {
		t.Fatalf("write run A: %v", err)
	}
	if err := writeJSON(pathB, runB); err != nil {
		t.Fatalf("write run B: %v", err)
	}

	report, err := CompareRuns(repo, "a", "b")
	if err != nil {
		t.Fatalf("compare runs: %v", err)
	}
	if report.Delta.Score <= 0 {
		t.Fatalf("expected positive score delta, got %+v", report.Delta)
	}
	if report.Delta.PassDelta != 1 {
		t.Fatalf("expected pass delta 1, got %+v", report.Delta)
	}
}
