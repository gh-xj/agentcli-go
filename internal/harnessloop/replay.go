package harnessloop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ReplayReport struct {
	SchemaVersion  string         `json:"schema_version"`
	RunID          string         `json:"run_id"`
	Iteration      int            `json:"iteration"`
	Baseline       ScenarioResult `json:"baseline"`
	Replay         ScenarioResult `json:"replay"`
	ReplayJudge    JudgeScore     `json:"replay_judge"`
	ReplayFindings []Finding      `json:"replay_findings"`
}

type replayContext struct {
	Scenario ScenarioResult `json:"scenario"`
}

func ReplayIteration(repoRoot, runID string, iteration int, threshold float64) (ReplayReport, error) {
	if runID == "" {
		return ReplayReport{}, fmt.Errorf("run id is required")
	}
	if iteration <= 0 {
		return ReplayReport{}, fmt.Errorf("iteration must be >= 1")
	}
	ctxPath := filepath.Join(repoRoot, ".docs", "onboarding-loop", "runs", runID, fmt.Sprintf("iter-%02d", iteration), "planner-context.json")
	raw, err := os.ReadFile(ctxPath)
	if err != nil {
		return ReplayReport{}, fmt.Errorf("replay artifacts not found for run %s iter %d; run with 'agentops loop lab ... --verbose-artifacts' to enable replay", runID, iteration)
	}
	var ctx replayContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return ReplayReport{}, err
	}
	if len(ctx.Scenario.Steps) == 0 {
		return ReplayReport{}, fmt.Errorf("no scenario steps found in replay context")
	}

	steps := make([]Step, 0, len(ctx.Scenario.Steps))
	for _, s := range ctx.Scenario.Steps {
		steps = append(steps, Step{Name: s.Name, Command: s.Command})
	}
	replayScenario := Scenario{
		Name:        "replay-" + runID,
		Description: "replay of recorded iteration",
		Steps:       steps,
	}
	replay, err := RunScenario(replayScenario)
	if err != nil {
		return ReplayReport{}, err
	}
	findings := DetectFindings(replay)
	judge := Judge(replay, findings, threshold)

	return ReplayReport{
		SchemaVersion:  "v1",
		RunID:          runID,
		Iteration:      iteration,
		Baseline:       ctx.Scenario,
		Replay:         replay,
		ReplayJudge:    judge,
		ReplayFindings: findings,
	}, nil
}
