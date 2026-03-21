package resource

import agentcli "github.com/gh-xj/agentcli-go"

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
	Create(ctx *agentcli.AppContext, slug string, opts map[string]string) (*Record, error)
	List(ctx *agentcli.AppContext, filter Filter) ([]Record, error)
	Get(ctx *agentcli.AppContext, id string) (*Record, error)
}

// Validator is an optional interface for resources that support compliance checks.
type Validator interface {
	Validate(ctx *agentcli.AppContext, id string) (*agentcli.DoctorReport, error)
}

// Deleter is an optional interface for resources that support deletion.
type Deleter interface {
	Delete(ctx *agentcli.AppContext, id string) error
}

// Syncer is an optional interface for resources that support external sync.
type Syncer interface {
	Sync(ctx *agentcli.AppContext, id string) error
}

// Transitioner is an optional interface for resources with state machines.
type Transitioner interface {
	Transition(ctx *agentcli.AppContext, id string, action string) (*Record, error)
}
