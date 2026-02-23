package harnessloop

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReportsCleansOldRunsAndWritesReview(t *testing.T) {
	repo := t.TempDir()
	runs := filepath.Join(repo, ".docs", "onboarding-loop", "runs")
	for i := 1; i <= 25; i++ {
		if err := os.MkdirAll(filepath.Join(runs, fmt.Sprintf("20260101-0000%02d", i)), 0755); err != nil {
			t.Fatalf("mkdir run: %v", err)
		}
	}
	result := RunResult{SchemaVersion: "v1", RunID: "20260223-999999", Mode: "committee", Judge: JudgeScore{Score: 9.9, Threshold: 9.0, Pass: true}}
	if err := WriteReports(repo, result); err != nil {
		t.Fatalf("write reports: %v", err)
	}
	entries, err := os.ReadDir(runs)
	if err != nil {
		t.Fatalf("read runs: %v", err)
	}
	if len(entries) > 20 {
		t.Fatalf("expected at most 20 run dirs, got %d", len(entries))
	}
	if _, err := os.Stat(filepath.Join(repo, ".docs", "onboarding-loop", "maintainer", "latest-review.md")); err != nil {
		t.Fatalf("missing review latest: %v", err)
	}
}
