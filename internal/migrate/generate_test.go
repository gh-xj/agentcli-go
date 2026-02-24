package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateSafeModeWritesParallelWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	plan := Plan{
		SchemaVersion: "v1",
		RepoRoot:      repoRoot,
		Source:        "scripts",
		Scripts: []ScriptPlan{
			{
				Script: ScriptInfo{
					Name:       "sync-data",
					Path:       "scripts/sync-data.sh",
					Shell:      ShellSh,
					Executable: true,
				},
				Strategy: StrategyAuto,
			},
		},
		Summary: PlanSummary{
			Total: 1,
			Auto:  1,
		},
	}

	result, err := Generate(plan, GenerateOptions{
		Mode:  ModeSafe,
		Apply: true,
	})
	if err != nil {
		t.Fatalf("generate migration output: %v", err)
	}

	if result.OutputRoot == repoRoot {
		t.Fatalf("expected safe mode to write to parallel workspace, got %s", result.OutputRoot)
	}
	if !filepath.IsAbs(result.OutputRoot) {
		t.Fatalf("expected absolute output root, got %s", result.OutputRoot)
	}

	checkExists := func(rel string) {
		t.Helper()
		path := filepath.Join(result.OutputRoot, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", rel, err)
		}
	}
	checkExists(filepath.Join("cmd", "sync-data.go"))
	checkExists(filepath.Join("docs", "migration", "report.md"))
	checkExists(filepath.Join("docs", "migration", "report.json"))
	checkExists(filepath.Join("docs", "migration", "plan.json"))
	checkExists(filepath.Join("docs", "migration", "compatibility.md"))
	checkExists("skill.migration.md")
}
