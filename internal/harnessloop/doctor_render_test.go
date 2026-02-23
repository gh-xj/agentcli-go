package harnessloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderDoctorMarkdownGolden(t *testing.T) {
	r := DoctorReport{
		SchemaVersion:    "v1",
		LeanReady:        true,
		LabFeaturesReady: false,
		Findings:         nil,
		Suggestions:      []string{"Lean path ready. Use 'agentcli loop judge' for daily checks."},
		ReviewPath:       ".docs/onboarding-loop/maintainer/latest-review.md",
	}
	got := RenderDoctorMarkdown(r)
	goldenPath := filepath.Join("testdata", "doctor.md.golden")
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(wantBytes) {
		t.Fatalf("doctor markdown drift\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}
