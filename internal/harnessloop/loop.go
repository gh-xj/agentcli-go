package harnessloop

import (
	"fmt"
	"time"
)

type Config struct {
	RepoRoot       string
	Threshold      float64
	MaxIterations  int
	AutoFix        bool
	AutoCommit     bool
	Branch         string
	Mode           string
	RoleConfigPath string
	Seed           int64
	Budget         int
}

func RunLoop(cfg Config) (RunResult, error) {
	cfg = normalizeConfig(cfg)

	started := time.Now().UTC()
	runID := started.Format("20060102-150405")
	agentcliBin, err := BuildLocalAgentCLIBinary(cfg.RepoRoot)
	if err != nil {
		return RunResult{}, err
	}

	var result RunResult
	switch cfg.Mode {
	case "committee":
		result, err = runCommittee(cfg, agentcliBin, started, runID)
	default:
		result, err = runClassic(cfg, agentcliBin, started, runID)
	}
	if err != nil {
		return result, err
	}

	if err := WriteReports(cfg.RepoRoot, result); err != nil {
		return result, err
	}

	if cfg.AutoCommit {
		if err := EnsureBranch(cfg.RepoRoot, cfg.Branch); err != nil {
			return result, err
		}
		committed, err := CommitIfDirty(cfg.RepoRoot, fmt.Sprintf("chore: onboarding loop %s score %.2f", result.Mode, result.Judge.Score))
		if err != nil {
			return result, err
		}
		if committed {
			result.Branch = cfg.Branch
		}
	}
	return result, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 9.0
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 3
	}
	if cfg.Branch == "" {
		cfg.Branch = "autofix/onboarding-loop"
	}
	if cfg.Mode == "" {
		cfg.Mode = "classic"
	}
	if cfg.Budget <= 0 {
		cfg.Budget = 1
	}
	return cfg
}

func runClassic(cfg Config, agentcliBin string, started time.Time, runID string) (RunResult, error) {
	var best RunResult
	best.Judge.Score = -1
	best.Mode = cfg.Mode
	best.RunID = runID

	iterations := 0
	for ; iterations < cfg.MaxIterations; iterations++ {
		scenario := DefaultOnboardingScenario(agentcliBin)
		sr, err := RunScenario(scenario)
		if err != nil {
			return RunResult{}, err
		}
		findings := DetectFindings(sr)
		findings = append(findings, CheckOnboardingInstallReadiness(cfg.RepoRoot)...)
		judge := Judge(sr, findings, cfg.Threshold)
		run := RunResult{
			SchemaVersion: "v1",
			StartedAt:     started,
			FinishedAt:    time.Now().UTC(),
			Scenario:      sr,
			Findings:      findings,
			Judge:         judge,
			Iterations:    iterations + 1,
			Branch:        CurrentBranch(cfg.RepoRoot),
			Mode:          cfg.Mode,
			RunID:         runID,
		}
		if judge.Score > best.Judge.Score {
			best = run
		}
		if judge.Pass || !cfg.AutoFix {
			break
		}
		applied, err := ApplyFixes(cfg.RepoRoot, findings)
		if err != nil {
			return run, err
		}
		run.FixesApplied = append(run.FixesApplied, applied...)
		best = run
		if len(applied) == 0 {
			break
		}
	}
	best.Iterations = iterations + 1
	best.FinishedAt = time.Now().UTC()
	return best, nil
}
