package migrate

import "testing"

func TestPlanClassifiesScriptStrategies(t *testing.T) {
	scan := ScanResult{
		RepoRoot: "/tmp/repo",
		Source:   "scripts",
		Scripts: []ScriptInfo{
			{
				Name:        "simple-sync",
				Path:        "scripts/simple-sync.sh",
				Shell:       ShellSh,
				Executable:  true,
				UsesArgs:    true,
				UsesEnv:     true,
				SizeBytes:   120,
				RiskSignals: nil,
			},
			{
				Name:        "medium-wrap",
				Path:        "scripts/medium-wrap.sh",
				Shell:       ShellBash,
				Executable:  true,
				UsesArgs:    true,
				UsesEnv:     true,
				SizeBytes:   640,
				RiskSignals: []string{"trap"},
			},
			{
				Name:        "complex-manual",
				Path:        "scripts/complex-manual.sh",
				Shell:       ShellBash,
				Executable:  true,
				UsesArgs:    true,
				UsesEnv:     true,
				SizeBytes:   2048,
				RiskSignals: []string{"eval"},
			},
		},
	}

	plan := BuildPlan(scan)
	if len(plan.Scripts) != 3 {
		t.Fatalf("unexpected plan size: got %d want %d", len(plan.Scripts), 3)
	}
	if plan.Scripts[0].Strategy != StrategyAuto {
		t.Fatalf("unexpected simple script strategy: %+v", plan.Scripts[0])
	}
	if plan.Scripts[1].Strategy != StrategyWrapper {
		t.Fatalf("unexpected medium script strategy: %+v", plan.Scripts[1])
	}
	if plan.Scripts[2].Strategy != StrategyManual {
		t.Fatalf("unexpected complex script strategy: %+v", plan.Scripts[2])
	}
	if plan.Summary.Auto != 1 || plan.Summary.Wrapper != 1 || plan.Summary.Manual != 1 {
		t.Fatalf("unexpected summary: %+v", plan.Summary)
	}
}
