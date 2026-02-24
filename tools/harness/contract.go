package harness

import "time"

const SummarySchemaVersion = "harness.v1"

type Status string

const (
	StatusOK   Status = "ok"
	StatusFail Status = "fail"
)

type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

type Artifact struct {
	Name string `json:"name,omitempty"`
	Kind string `json:"kind,omitempty"`
	Path string `json:"path,omitempty"`
}

type Failure struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Hint      string `json:"hint,omitempty"`
	Retryable bool   `json:"retryable"`
}

type CommandSummary struct {
	SchemaVersion string        `json:"schema_version"`
	Command       string        `json:"command"`
	Status        string        `json:"status"`
	StartedAt     time.Time     `json:"started_at"`
	FinishedAt    time.Time     `json:"finished_at"`
	DurationMs    int64         `json:"duration_ms"`
	Checks        []CheckResult `json:"checks,omitempty"`
	Failures      []Failure     `json:"failures,omitempty"`
	Artifacts     []Artifact    `json:"artifacts,omitempty"`
	Data          any           `json:"data,omitempty"`
}

type CommandOutcome struct {
	Checks    []CheckResult
	Failures  []Failure
	Artifacts []Artifact
	Data      any
}
