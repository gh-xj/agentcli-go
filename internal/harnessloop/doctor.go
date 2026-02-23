package harnessloop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DoctorReport struct {
	SchemaVersion    string    `json:"schema_version"`
	LeanReady        bool      `json:"lean_ready"`
	LabFeaturesReady bool      `json:"lab_features_ready"`
	Findings         []Finding `json:"findings"`
	Suggestions      []string  `json:"suggestions"`
	ReviewPath       string    `json:"review_path"`
}

func LoopDoctor(repoRoot string) DoctorReport {
	findings := CheckOnboardingInstallReadiness(repoRoot)
	reviewPath := filepath.Join(repoRoot, ".docs", "onboarding-loop", "maintainer", "latest-review.md")
	labReady := hasAnyIterArtifacts(repoRoot)
	suggestions := []string{}
	if len(findings) > 0 {
		suggestions = append(suggestions, "Fix onboarding install prompt issues before relying on loop scores.")
	}
	if !labReady {
		suggestions = append(suggestions, "Run 'agentcli loop lab run --verbose-artifacts --max-iterations 1' to enable replay/forensics.")
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Lean path ready. Use 'agentcli loop judge' for daily checks.")
	}
	return DoctorReport{
		SchemaVersion:    "v1",
		LeanReady:        len(findings) == 0,
		LabFeaturesReady: labReady,
		Findings:         findings,
		Suggestions:      suggestions,
		ReviewPath:       reviewPath,
	}
}

func RenderDoctorMarkdown(r DoctorReport) string {
	var b strings.Builder
	b.WriteString("# Loop Doctor\n\n")
	b.WriteString(fmt.Sprintf("- Lean ready: `%v`\n", r.LeanReady))
	b.WriteString(fmt.Sprintf("- Lab features ready: `%v`\n", r.LabFeaturesReady))
	b.WriteString(fmt.Sprintf("- Review path: `%s`\n", r.ReviewPath))
	b.WriteString("\n## Findings\n\n")
	if len(r.Findings) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, f := range r.Findings {
			b.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", f.Code, f.Message, f.Source))
		}
	}
	b.WriteString("\n## Suggestions\n\n")
	for _, s := range r.Suggestions {
		b.WriteString(fmt.Sprintf("- %s\n", s))
	}
	return b.String()
}

func hasAnyIterArtifacts(repoRoot string) bool {
	runsDir := filepath.Join(repoRoot, ".docs", "onboarding-loop", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return false
	}
	for _, run := range entries {
		if !run.IsDir() {
			continue
		}
		iterEntries, err := os.ReadDir(filepath.Join(runsDir, run.Name()))
		if err != nil {
			continue
		}
		for _, ie := range iterEntries {
			if ie.IsDir() && strings.HasPrefix(ie.Name(), "iter-") {
				return true
			}
		}
	}
	return false
}
