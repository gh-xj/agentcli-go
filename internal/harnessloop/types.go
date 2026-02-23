package harnessloop

import "time"

type Step struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type Scenario struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	WorkDir     string `json:"work_dir"`
	Steps       []Step `json:"steps"`
}

type StepResult struct {
	Name         string `json:"name"`
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	DurationMs   int64  `json:"duration_ms"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	CombinedTail string `json:"combined_tail"`
}

type ScenarioResult struct {
	Name       string       `json:"name"`
	StartedAt  time.Time    `json:"started_at"`
	FinishedAt time.Time    `json:"finished_at"`
	OK         bool         `json:"ok"`
	Steps      []StepResult `json:"steps"`
}

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

type RunResult struct {
	SchemaVersion string         `json:"schema_version"`
	StartedAt     time.Time      `json:"started_at"`
	FinishedAt    time.Time      `json:"finished_at"`
	Scenario      ScenarioResult `json:"scenario"`
	Findings      []Finding      `json:"findings"`
	Judge         JudgeScore     `json:"judge"`
	Iterations    int            `json:"iterations"`
	Branch        string         `json:"branch"`
	FixesApplied  []string       `json:"fixes_applied"`
	Mode          string         `json:"mode,omitempty"`
	RunID         string         `json:"run_id,omitempty"`
	Committee     *CommitteeMeta `json:"committee,omitempty"`
}

type JudgeScore struct {
	Score                float64 `json:"score"`
	Threshold            float64 `json:"threshold"`
	Pass                 bool    `json:"pass"`
	UXScore              float64 `json:"ux_score"`
	QualityScore         float64 `json:"quality_score"`
	PenaltyScore         float64 `json:"penalty_score"`
	ScenarioPassRate     float64 `json:"scenario_pass_rate"`
	CounterIntuitiveFind int     `json:"counter_intuitive_findings"`
	HardFailures         int     `json:"hard_failures"`
}

type CommitteeMeta struct {
	Planner RoleExecution `json:"planner"`
	Fixer   RoleExecution `json:"fixer"`
	Judger  RoleExecution `json:"judger"`
}

type RoleExecution struct {
	Strategy    string   `json:"strategy"`
	Independent bool     `json:"independent,omitempty"`
	Command     string   `json:"command,omitempty"`
	Artifacts   string   `json:"artifacts,omitempty"`
	Applied     []string `json:"applied,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	ExitCode    int      `json:"exit_code,omitempty"`
	StderrTail  string   `json:"stderr_tail,omitempty"`
}

type RoleConfig struct {
	Planner RoleSpec `json:"planner"`
	Fixer   RoleSpec `json:"fixer"`
	Judger  RoleSpec `json:"judger"`
}

type RoleSpec struct {
	Strategy string `json:"strategy"`
	Command  string `json:"command"`
}
