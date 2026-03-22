package resource

import agentops "github.com/gh-xj/agentops"

// Record is a generic representation of a resource instance.
type Record struct {
	Kind    string         `json:"kind"`
	ID      string         `json:"id"`
	Fields  map[string]any `json:"fields"`
	RawPath string         `json:"raw_path,omitempty"`
}

// Filter constrains which records are returned by List.
type Filter map[string]string

// ResourceSchema describes the shape and rules of a resource kind.
type ResourceSchema struct {
	Kind        string
	Fields      []FieldDef
	Statuses    []string
	CreateArgs  []ArgDef
	Description string
}

// FieldDef describes one field in a resource schema.
type FieldDef struct {
	Name     string
	Type     string
	Required bool
}

// ArgDef describes one argument accepted by Create.
type ArgDef struct {
	Name        string
	Description string
	Required    bool
}

// Resource is the core interface every agentops resource kind must implement.
type Resource interface {
	Schema() ResourceSchema
	Create(ctx *agentops.AppContext, slug string, opts map[string]string) (*Record, error)
	List(ctx *agentops.AppContext, filter Filter) ([]Record, error)
	Get(ctx *agentops.AppContext, id string) (*Record, error)
}

// Validator is an optional interface for resources that support compliance checks.
type Validator interface {
	Validate(ctx *agentops.AppContext, id string) (*agentops.DoctorReport, error)
}

// Deleter is an optional interface for resources that support deletion.
type Deleter interface {
	Delete(ctx *agentops.AppContext, id string) error
}

// Syncer is an optional interface for resources that support external sync.
type Syncer interface {
	Sync(ctx *agentops.AppContext, id string) error
}

// Transitioner is an optional interface for resources with state machines.
type Transitioner interface {
	Transition(ctx *agentops.AppContext, id string, action string) (*Record, error)
}

// Doctor is an optional interface for resources that support health checks.
type Doctor interface {
	Doctor(ctx *agentops.AppContext) ([]DoctorCheck, error)
}

// Pruner is an optional interface for resources that support cleanup of stale entries.
type Pruner interface {
	Prune(ctx *agentops.AppContext, confirm bool) ([]PruneResult, error)
}

// DoctorCheck represents a single health check finding.
type DoctorCheck struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // ok, warn, err
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// PruneResult represents a single cleanup action taken or proposed.
type PruneResult struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Action string `json:"action"` // removed, would_remove, skipped
	Reason string `json:"reason"`
}
