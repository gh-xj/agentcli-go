package harness

type FlagSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

type ArtifactSpec struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	PathHint    string `json:"path_hint,omitempty"`
	Description string `json:"description,omitempty"`
}

type CommandCapability struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Flags       []FlagSpec     `json:"flags,omitempty"`
	Artifacts   []ArtifactSpec `json:"artifacts,omitempty"`
}

type Capabilities struct {
	SchemaVersion string              `json:"schema_version"`
	GlobalFlags   []FlagSpec          `json:"global_flags"`
	Commands      []CommandCapability `json:"commands"`
}

func DefaultCapabilities() Capabilities {
	global := []FlagSpec{
		{
			Name:        "--format",
			Type:        "enum(text|json|ndjson)",
			Default:     "text",
			Description: "output renderer",
		},
		{
			Name:        "--summary",
			Type:        "path",
			Description: "write summary json to file",
		},
		{
			Name:        "--no-color",
			Type:        "bool",
			Default:     "false",
			Description: "disable colorized output",
		},
		{
			Name:        "--dry-run",
			Type:        "bool",
			Default:     "false",
			Description: "plan command without making changes",
		},
		{
			Name:        "--explain",
			Type:        "bool",
			Default:     "false",
			Description: "include operator guidance in summary data",
		},
	}

	return Capabilities{
		SchemaVersion: SummarySchemaVersion,
		GlobalFlags:   global,
		Commands: []CommandCapability{
			{
				Name:        "capabilities",
				Description: "print machine-readable command capabilities",
			},
			{
				Name:        "doctor",
				Description: "check loop readiness",
				Flags: []FlagSpec{
					{Name: "--repo-root", Type: "path", Default: "."},
				},
			},
			{
				Name:        "profiles",
				Description: "list loop profiles",
				Flags: []FlagSpec{
					{Name: "--repo-root", Type: "path", Default: "."},
				},
			},
			{
				Name:        "profile",
				Description: "run loop using a named profile",
				Flags: []FlagSpec{
					{Name: "<name>", Type: "string", Required: true},
					{Name: "--repo-root", Type: "path", Default: "."},
					{Name: "--threshold", Type: "float"},
					{Name: "--max-iterations", Type: "int"},
					{Name: "--branch", Type: "string"},
					{Name: "--api", Type: "url"},
					{Name: "--role-config", Type: "path"},
					{Name: "--verbose-artifacts", Type: "bool"},
					{Name: "--no-verbose-artifacts", Type: "bool"},
				},
				Artifacts: []ArtifactSpec{
					{Name: "loop-run", Kind: "json", PathHint: ".docs/onboarding-loop/runs/<run-id>/run-result.json"},
				},
			},
			{
				Name:        "quality",
				Description: "run the default quality profile",
				Flags: []FlagSpec{
					{Name: "--repo-root", Type: "path", Default: "."},
					{Name: "--threshold", Type: "float"},
					{Name: "--max-iterations", Type: "int"},
					{Name: "--branch", Type: "string"},
					{Name: "--api", Type: "url"},
					{Name: "--role-config", Type: "path"},
					{Name: "--verbose-artifacts", Type: "bool"},
					{Name: "--no-verbose-artifacts", Type: "bool"},
				},
				Artifacts: []ArtifactSpec{
					{Name: "loop-run", Kind: "json", PathHint: ".docs/onboarding-loop/runs/<run-id>/run-result.json"},
				},
			},
			{
				Name:        "regression",
				Description: "validate behavior snapshot against baseline",
				Flags: []FlagSpec{
					{Name: "--repo-root", Type: "path", Default: "."},
					{Name: "--profile", Type: "string", Default: "quality"},
					{Name: "--baseline", Type: "path"},
					{Name: "--write-baseline", Type: "bool"},
				},
				Artifacts: []ArtifactSpec{
					{Name: "behavior-baseline", Kind: "json", PathHint: "testdata/regression/loop-<profile>.behavior-baseline.json"},
				},
			},
			{
				Name:        "lab",
				Description: "advanced loop operations",
				Flags: []FlagSpec{
					{Name: "<action>", Type: "enum(compare|replay|run|judge|autofix)", Required: true},
					{Name: "--repo-root", Type: "path", Default: "."},
				},
			},
		},
	}
}
