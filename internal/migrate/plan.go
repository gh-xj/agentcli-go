package migrate

import (
	"slices"
	"time"
)

func BuildPlan(scan ScanResult) Plan {
	plan := Plan{
		SchemaVersion: "v1",
		RepoRoot:      scan.RepoRoot,
		Source:        scan.Source,
		GeneratedAt:   time.Now().UTC(),
		Scripts:       make([]ScriptPlan, 0, len(scan.Scripts)),
		Summary: PlanSummary{
			Total: len(scan.Scripts),
		},
	}

	for _, script := range scan.Scripts {
		strategy, reasons, score := classifyScript(script)
		scriptPlan := ScriptPlan{
			Script:   script,
			Strategy: strategy,
			Reasons:  reasons,
			Score:    score,
		}
		plan.Scripts = append(plan.Scripts, scriptPlan)
		switch strategy {
		case StrategyAuto:
			plan.Summary.Auto++
		case StrategyWrapper:
			plan.Summary.Wrapper++
		default:
			plan.Summary.Manual++
		}
	}

	return plan
}

func classifyScript(script ScriptInfo) (MigrationStrategy, []string, float64) {
	reasons := make([]string, 0, 3)
	if script.Shell != ShellSh && script.Shell != ShellBash {
		return StrategyManual, []string{"unsupported_shell"}, 0.1
	}
	if slices.Contains(script.RiskSignals, "eval") {
		return StrategyManual, []string{"contains_eval"}, 0.1
	}
	if slices.Contains(script.RiskSignals, "source") {
		reasons = append(reasons, "contains_source")
	}
	if slices.Contains(script.RiskSignals, "trap") {
		reasons = append(reasons, "contains_trap")
	}
	if slices.Contains(script.RiskSignals, "heredoc") {
		reasons = append(reasons, "contains_heredoc")
	}
	if len(reasons) > 0 || script.SizeBytes > 1024 {
		if script.SizeBytes > 1024 {
			reasons = append(reasons, "large_script")
		}
		return StrategyWrapper, reasons, 0.6
	}

	return StrategyAuto, nil, 0.95
}
