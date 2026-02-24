package migrate

import "time"

type ShellType string

const (
	ShellUnknown ShellType = "unknown"
	ShellSh      ShellType = "sh"
	ShellBash    ShellType = "bash"
)

type MigrationMode string

const (
	ModeSafe    MigrationMode = "safe"
	ModeInPlace MigrationMode = "in-place"
)

type ScriptInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Shell        ShellType `json:"shell"`
	Executable   bool      `json:"executable"`
	UsesArgs     bool      `json:"uses_args"`
	UsesEnv      bool      `json:"uses_env"`
	ExternalDeps []string  `json:"external_deps,omitempty"`
	RiskSignals  []string  `json:"risk_signals,omitempty"`
	SizeBytes    int       `json:"size_bytes"`
}

type ScanResult struct {
	RepoRoot string       `json:"repo_root"`
	Source   string       `json:"source"`
	Scanned  int          `json:"scanned"`
	Scripts  []ScriptInfo `json:"scripts"`
}

type MigrationStrategy string

const (
	StrategyAuto    MigrationStrategy = "auto"
	StrategyWrapper MigrationStrategy = "wrapper"
	StrategyManual  MigrationStrategy = "manual"
)

type ScriptPlan struct {
	Script   ScriptInfo        `json:"script"`
	Strategy MigrationStrategy `json:"strategy"`
	Reasons  []string          `json:"reasons,omitempty"`
	Score    float64           `json:"score"`
}

type Plan struct {
	SchemaVersion string       `json:"schema_version"`
	RepoRoot      string       `json:"repo_root"`
	Source        string       `json:"source"`
	GeneratedAt   time.Time    `json:"generated_at"`
	Scripts       []ScriptPlan `json:"scripts"`
	Summary       PlanSummary  `json:"summary"`
}

type PlanSummary struct {
	Total   int `json:"total"`
	Auto    int `json:"auto"`
	Wrapper int `json:"wrapper"`
	Manual  int `json:"manual"`
}

type Report struct {
	SchemaVersion string       `json:"schema_version"`
	RepoRoot      string       `json:"repo_root"`
	Source        string       `json:"source"`
	Summary       PlanSummary  `json:"summary"`
	Scripts       []ScriptPlan `json:"scripts"`
}
