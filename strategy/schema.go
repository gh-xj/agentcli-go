package strategy

// Strategy holds the fully loaded .agentops/ configuration.
type Strategy struct {
	Root           string // absolute path to project root (parent of .agentops/)
	Storage        StorageConfig
	Transitions    TransitionsConfig
	Risk           map[string]any
	Routing        map[string]any
	Budget         map[string]any
	Hooks          HooksConfig
	SchemaTemplate string // raw content of schema.md
}

// StorageConfig controls where case records are stored.
type StorageConfig struct {
	Backend      string `yaml:"backend"`        // "separate-repo" or "in-repo"
	CaseRepoPath string `yaml:"case_repo_path"` // relative path to case repo
}

// TransitionsConfig defines the state machine for case lifecycle.
type TransitionsConfig struct {
	Categories  map[string][]string      `yaml:"categories"`
	Initial     string                   `yaml:"initial"`
	Transitions map[string]TransitionDef `yaml:"transitions"`
}

// TransitionDef describes one allowed state transition.
type TransitionDef struct {
	From any    `yaml:"from"` // string or []string
	To   string `yaml:"to"`
}

// FromStates returns the from states as a string slice.
func (t TransitionDef) FromStates() []string {
	switch v := t.From.(type) {
	case string:
		return []string{v}
	case []any:
		result := make([]string, len(v))
		for i, s := range v {
			result[i] = s.(string)
		}
		return result
	}
	return nil
}

// HooksConfig defines pre/post lifecycle hooks.
type HooksConfig struct {
	PreDispatch []string `yaml:"pre_dispatch"`
	PostClose   []string `yaml:"post_close"`
}
