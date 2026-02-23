package harnessloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultRunRetention = 20

func WriteReports(repoRoot string, result RunResult) error {
	dir := filepath.Join(repoRoot, ".docs", "onboarding-loop")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	latest := filepath.Join(dir, "latest-summary.json")
	if err := writeJSON(latest, result); err != nil {
		return err
	}

	if err := writeReviewLatest(repoRoot, result); err != nil {
		return err
	}

	if err := cleanupRunArtifacts(repoRoot, defaultRunRetention); err != nil {
		return err
	}
	return nil
}

func writeReviewLatest(repoRoot string, result RunResult) error {
	reviewDir := filepath.Join(repoRoot, ".docs", "onboarding-loop", "maintainer")
	if err := os.MkdirAll(reviewDir, 0755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Verification Review\n\n")
	b.WriteString(fmt.Sprintf("- Run ID: `%s`\n", result.RunID))
	b.WriteString(fmt.Sprintf("- Mode: `%s`\n", result.Mode))
	b.WriteString(fmt.Sprintf("- Score: `%.2f/10` (threshold `%.2f`)\n", result.Judge.Score, result.Judge.Threshold))
	b.WriteString(fmt.Sprintf("- Pass: `%v`\n", result.Judge.Pass))
	b.WriteString(fmt.Sprintf("- Iterations: `%d`\n", result.Iterations))
	b.WriteString(fmt.Sprintf("- Branch: `%s`\n", result.Branch))
	b.WriteString("\n## Findings\n\n")
	if len(result.Findings) == 0 {
		b.WriteString("- none\n")
	} else {
		limit := len(result.Findings)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			f := result.Findings[i]
			b.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", f.Code, f.Message, f.Source))
		}
		if len(result.Findings) > limit {
			b.WriteString(fmt.Sprintf("- ... plus %d more\n", len(result.Findings)-limit))
		}
	}
	if result.Committee != nil {
		b.WriteString("\n## Committee\n\n")
		b.WriteString(fmt.Sprintf("- Planner score: `%.2f`\n", result.Judge.PlannerScore))
		b.WriteString(fmt.Sprintf("- Fixer score: `%.2f`\n", result.Judge.FixerScore))
		b.WriteString(fmt.Sprintf("- Judger score: `%.2f`\n", result.Judge.JudgerScore))
	}
	return os.WriteFile(filepath.Join(reviewDir, "latest-review.md"), []byte(b.String()), 0644)
}

func cleanupRunArtifacts(repoRoot string, keep int) error {
	if keep <= 0 {
		keep = defaultRunRetention
	}
	runsDir := filepath.Join(repoRoot, ".docs", "onboarding-loop", "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	if len(dirs) <= keep {
		return nil
	}
	for _, old := range dirs[:len(dirs)-keep] {
		if err := os.RemoveAll(filepath.Join(runsDir, old)); err != nil {
			return err
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
