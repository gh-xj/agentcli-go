package dogfood

import "time"

type EventType string

const (
	EventTypeCIFailure    EventType = "ci_failure"
	EventTypeRuntimeError EventType = "runtime_error"
	EventTypeDocsDrift    EventType = "docs_drift"
)

type Event struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	EventType     EventType `json:"event_type"`
	SignalSource  string    `json:"signal_source"`
	Timestamp     time.Time `json:"timestamp"`
	RepoGuess     string    `json:"repo_guess,omitempty"`
	ErrorSummary  string    `json:"error_summary,omitempty"`
	EvidencePaths []string  `json:"evidence_paths,omitempty"`
}

type Action string

const (
	ActionCreateIssue   Action = "create_issue"
	ActionAppendComment Action = "append_comment"
	ActionPendingReview Action = "pending_review"
	ActionQueueRetry    Action = "queue_retry"
)

type DecisionInput struct {
	RepoConfidence float64
	HasOpenIssue   bool
	Fingerprint    string
}

type Decision struct {
	Action Action
	Reason string
}
