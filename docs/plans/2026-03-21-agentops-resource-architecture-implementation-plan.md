# agentops Resource Architecture Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge casectl into agentcli-go, rename binary to `agentops`, introduce Resource abstraction with auto-generated CLI commands.

**Architecture:** Resource interface as core primitive. Resources (case, slot, project) implement uniform CRUD. cobrax auto-generates noun-verb Cobra commands from a registry. Strategy package loads `.agentops/` config. Dal layer shared by all resources.

**Tech Stack:** Go 1.25.5, Cobra, gopkg.in/yaml.v3 (new), itchyny/gojq (new), golang.org/x/term (new), zerolog

**Spec:** `docs/plans/2026-03-21-agentops-resource-architecture-design.md`

**Deviations from spec:**
- Exit codes start at 10 (not 3) to avoid collision with existing ExitPreflightDependency/ExitRuntimeExternal
- Resource methods take `*agentcli.AppContext` directly (spec's `*agentcli.AppContext` wrapper dropped for simplicity)
- `strategy/` uses raw `os` calls (not `dal.FileSystem`) — acceptable because strategy loading happens once at startup and tests use real temp dirs
- `cobrax.Execute()` old signature kept as-is; new function named `ExecuteRoot()` to avoid breaking existing examples

---

## Prerequisite: Dependency Setup

Before any task, run in the repo root:

```bash
cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go
go get gopkg.in/yaml.v3@latest
go get github.com/itchyny/gojq@latest
go get golang.org/x/term@latest
go mod tidy
git add go.mod go.sum
git commit -m "deps: add yaml.v3, gojq, x/term"
```

---

## File Structure

### New Files

| File | Responsibility |
|---|---|
| `resource/resource.go` | Resource, Record, Filter, Schema types + optional interfaces |
| `resource/registry.go` | Registry (register, lookup, iterate) |
| `resource/registry_test.go` | Registry unit tests |
| `resource/case/case.go` | Case resource: Create, List, Get, Validate, Transition |
| `resource/case/case_test.go` | Case resource unit tests |
| `resource/case/frontmatter.go` | YAML frontmatter parse/write |
| `resource/case/frontmatter_test.go` | Frontmatter unit tests |
| `resource/case/transitions.go` | State machine loading + enforcement |
| `resource/case/transitions_test.go` | Transition unit tests |
| `resource/slot/slot.go` | Slot resource: Create, List, Get, Delete, Sync |
| `resource/slot/slot_test.go` | Slot resource unit tests |
| `resource/slot/worktree.go` | Git worktree operations |
| `resource/project/project.go` | Project resource: Create (scaffold) |
| `resource/project/templates.go` | Embedded Go templates (from service/templates.go) |
| `strategy/loader.go` | Walk-up discovery, parse all YAML, load defaults |
| `strategy/loader_test.go` | Strategy loading unit tests |
| `strategy/schema.go` | Typed structs for each .agentops/ config file |
| `strategy/defaults/schema.md` | Default case record template |
| `strategy/defaults/storage.yaml` | Default storage config |
| `strategy/defaults/transitions.yaml` | Default state machine |
| `strategy/defaults/risk.yaml` | Default risk config |
| `strategy/defaults/routing.yaml` | Default routing config |
| `strategy/defaults/budget.yaml` | Default budget config |
| `strategy/defaults/hooks.yaml` | Default hooks config |
| `strategy/defaults/strategy.md` | Default strategy prose |
| `strategy/defaults/slot.md` | Default slot prose |
| `cobrax/resource_commands.go` | Auto-generate noun-verb commands from registry |
| `cobrax/resource_commands_test.go` | Command generation tests |
| `cobrax/render.go` | Output pipeline: table, JSON, jq, TSV |
| `cobrax/render_test.go` | Render unit tests |
| `cmd/agentops/main.go` | New entry point with composition root |
| `cmd/agentops/init.go` | `agentops init` standalone command |
| `cmd/agentops/doctor.go` | `agentops doctor` standalone command |
| `cmd/agentops/dispatch.go` | `agentops dispatch` standalone command |
| `cmd/agentops/loop.go` | `agentops loop` (migrated from cmd/agentcli) |
| `protocol/lifecycle.md` | Protocol docs (from casectl) |
| `protocol/record.md` | Protocol docs (from casectl) |
| `protocol/worker.md` | Protocol docs (from casectl) |
| `protocol/slot.md` | Protocol docs (from casectl) |
| `protocol/hooks.md` | Protocol docs (from casectl) |
| `schemas/case-record.schema.json` | JSON schema for case record output |
| `examples/case-tracking/.agentops/` | Example project with .agentops/ |
| `test/e2e/agentops_test.go` | E2E integration tests |

### Modified Files

| File | Change |
|---|---|
| `errors.go` | Add new exit codes (ExitStrategyMissing, ExitTransitionDenied, etc.) |
| `cobrax/cobrax.go` | Add BuildRoot(), change Execute() signature |
| `go.mod` | Add `itchyny/gojq` dependency |

### Deleted Files

| File | Reason |
|---|---|
| `cmd/agentcli/` (entire dir) | Replaced by `cmd/agentops/` |
| `operator/` (entire dir) | Replaced by `resource/` |
| `service/` (entire dir) | Replaced by resource registry + standalone commands |

---

## Chunk 1: Foundation Layer

### Task 1: Root Package — New Exit Codes

**Files:**
- Modify: `errors.go`
- Test: existing tests still pass

- [ ] **Step 1: Read current errors.go**

Read `/Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go/errors.go` to understand existing exit codes.

- [ ] **Step 2: Add new exit codes**

Add after existing constants in `errors.go`:

```go
const (
    ExitStrategyMissing   = 3 // no .agentops/ found
    ExitTransitionDenied  = 4 // invalid state transition
    ExitWorkerFailed      = 5 // worker returned error
    ExitValidationFailed  = 6 // case/strategy validation failed
)
```

Note: ExitPreflightDependency (3) and ExitRuntimeExternal (4) already exist. Reassign:
- Keep existing 3 and 4 as-is for backward compat of the root package
- Add new codes starting at 10 to avoid collision:

```go
const (
    ExitStrategyMissing   = 10
    ExitTransitionDenied  = 11
    ExitWorkerFailed      = 12
    ExitValidationFailed  = 13
)
```

- [ ] **Step 3: Run existing tests**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./...`
Expected: All existing tests pass.

- [ ] **Step 4: Commit**

```bash
git add errors.go
git commit -m "feat: add agentops-specific exit codes"
```

---

### Task 2: Resource Interface + Registry Package

**Files:**
- Create: `resource/resource.go`
- Create: `resource/registry.go`
- Create: `resource/registry_test.go`

- [ ] **Step 1: Write registry test**

```go
// resource/registry_test.go
package resource_test

import (
    "testing"

    "github.com/gh-xj/agentcli-go/resource"
)

type mockResource struct {
    schema resource.ResourceSchema
}

func (m *mockResource) Schema() resource.ResourceSchema { return m.schema }
func (m *mockResource) Create(_ *agentcli.AppContext, _ string, _ map[string]string) (*resource.Record, error) {
    return &resource.Record{Kind: m.schema.Kind, ID: "test-id"}, nil
}
func (m *mockResource) List(_ *agentcli.AppContext, _ resource.Filter) ([]resource.Record, error) {
    return nil, nil
}
func (m *mockResource) Get(_ *agentcli.AppContext, _ string) (*resource.Record, error) {
    return nil, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
    reg := resource.NewRegistry()
    mock := &mockResource{schema: resource.ResourceSchema{Kind: "test", Description: "test resource"}}
    reg.Register(mock)

    got, ok := reg.Get("test")
    if !ok {
        t.Fatal("expected to find registered resource")
    }
    if got.Schema().Kind != "test" {
        t.Errorf("got kind %q, want %q", got.Schema().Kind, "test")
    }
}

func TestRegistryGetMissing(t *testing.T) {
    reg := resource.NewRegistry()
    _, ok := reg.Get("missing")
    if ok {
        t.Fatal("expected not to find unregistered resource")
    }
}

func TestRegistryAllSorted(t *testing.T) {
    reg := resource.NewRegistry()
    reg.Register(&mockResource{schema: resource.ResourceSchema{Kind: "case"}})
    reg.Register(&mockResource{schema: resource.ResourceSchema{Kind: "slot"}})
    reg.Register(&mockResource{schema: resource.ResourceSchema{Kind: "project"}})

    all := reg.All()
    if len(all) != 3 {
        t.Fatalf("got %d resources, want 3", len(all))
    }
    if all[0].Schema().Kind != "case" || all[1].Schema().Kind != "project" || all[2].Schema().Kind != "slot" {
        t.Errorf("resources not sorted: %v", []string{all[0].Schema().Kind, all[1].Schema().Kind, all[2].Schema().Kind})
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./resource/...`
Expected: FAIL (package doesn't exist yet)

- [ ] **Step 3: Write resource.go**

```go
// resource/resource.go
package resource

import (
    agentcli "github.com/gh-xj/agentcli-go"
)

// Record is the universal output unit.
type Record struct {
    Kind    string         `json:"kind"`
    ID      string         `json:"id"`
    Fields  map[string]any `json:"fields"`
    RawPath string         `json:"raw_path,omitempty"`
}

// Filter is generic key-value. Each resource interprets its own keys.
type Filter map[string]string

// ResourceSchema declares what a resource looks like.
type ResourceSchema struct {
    Kind        string
    Fields      []FieldDef
    Statuses    []string
    CreateArgs  []ArgDef
    Description string
}

// FieldDef describes a field in a resource record.
type FieldDef struct {
    Name     string
    Type     string // "string", "datetime", "enum"
    Required bool
}

// ArgDef describes a positional argument for create.
type ArgDef struct {
    Name        string
    Description string
    Required    bool
}

// Resource is the core interface. Every manageable noun implements this.
// Create takes a user-provided slug (hint). The returned Record.ID is canonical.
type Resource interface {
    Schema() ResourceSchema
    Create(ctx Context, slug string, opts map[string]string) (*Record, error)
    List(ctx Context, filter Filter) ([]Record, error)
    Get(ctx Context, id string) (*Record, error)
}

// Optional capability interfaces. Commands only appear if implemented.

// Validator validates a resource by ID.
type Validator interface {
    Validate(ctx Context, id string) (*agentcli.DoctorReport, error)
}

// Deleter removes a resource by ID.
type Deleter interface {
    Delete(ctx Context, id string) error
}

// Syncer synchronizes a resource with its upstream.
type Syncer interface {
    Sync(ctx Context, id string) error
}

// Transitioner moves a resource through a state machine.
type Transitioner interface {
    Transition(ctx Context, id string, action string) (*Record, error)
}
```

- [ ] **Step 4: Write registry.go**

```go
// resource/registry.go
package resource

import "sort"

// Registry holds all registered resources.
type Registry struct {
    resources map[string]Resource
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
    return &Registry{resources: make(map[string]Resource)}
}

// Register adds a resource to the registry.
func (r *Registry) Register(res Resource) {
    r.resources[res.Schema().Kind] = res
}

// Get returns a resource by kind.
func (r *Registry) Get(kind string) (Resource, bool) {
    res, ok := r.resources[kind]
    return res, ok
}

// All returns all resources sorted by kind.
func (r *Registry) All() []Resource {
    keys := make([]string, 0, len(r.resources))
    for k := range r.resources {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    result := make([]Resource, len(keys))
    for i, k := range keys {
        result[i] = r.resources[k]
    }
    return result
}
```

- [ ] **Step 5: Run tests**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./resource/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add resource/
git commit -m "feat: add Resource interface and Registry"
```

---

### Task 3: Strategy Package

**Files:**
- Create: `strategy/schema.go`
- Create: `strategy/loader.go`
- Create: `strategy/loader_test.go`
- Create: `strategy/defaults/` (all embedded files)

- [ ] **Step 1: Write strategy schema types**

Create `strategy/schema.go` with typed structs for every `.agentops/` config file:

```go
// strategy/schema.go
package strategy

// Strategy holds the fully loaded .agentops/ configuration.
type Strategy struct {
    Root         string        // absolute path to project root (parent of .agentops/)
    Storage      StorageConfig
    Transitions  TransitionsConfig
    Risk         map[string]any
    Routing      map[string]any
    Budget       map[string]any
    Hooks        HooksConfig
    SchemaTemplate string      // raw content of schema.md
}

type StorageConfig struct {
    Backend      string `yaml:"backend"`       // "separate-repo" or "in-repo"
    CaseRepoPath string `yaml:"case_repo_path"` // relative path to case repo
}

type TransitionsConfig struct {
    Categories  map[string][]string `yaml:"categories"`
    Initial     string              `yaml:"initial"`
    Transitions map[string]TransitionDef `yaml:"transitions"`
}

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

type HooksConfig struct {
    PreDispatch []string `yaml:"pre_dispatch"`
    PostClose   []string `yaml:"post_close"`
}
```

- [ ] **Step 2: Create embedded default files**

Create `strategy/defaults/` directory with these files:

`strategy/defaults/transitions.yaml`:
```yaml
categories:
  active: [open, in_progress, blocked]
  completed: [resolved, closed_no_action]

initial: open

transitions:
  start:
    from: open
    to: in_progress
  block:
    from: [open, in_progress]
    to: blocked
  unblock:
    from: blocked
    to: in_progress
  resolve:
    from: [in_progress, blocked]
    to: resolved
  close_no_action:
    from: [open, blocked]
    to: closed_no_action
```

`strategy/defaults/storage.yaml`:
```yaml
backend: separate-repo
```

`strategy/defaults/schema.md`:
```markdown
---
type: intake
status: open
claimed_by: none
created: "YYYY-MM-DD"
---
# Case Title

## User Intent

## Findings

## Next Action

## Close Criteria
```

`strategy/defaults/strategy.md`:
```markdown
# Strategy

Project purpose and target repos.
```

`strategy/defaults/slot.md`:
```markdown
# Slot Convention

Slot names: lowercase alphanumeric and hyphens.
Path: ../worktrees/<project>-<slot>
```

`strategy/defaults/risk.yaml`:
```yaml
thresholds: {}
escalation: {}
```

`strategy/defaults/routing.yaml`:
```yaml
default_route: {}
overrides: {}
cues: {}
```

`strategy/defaults/budget.yaml`:
```yaml
limits: {}
tracking: {}
```

`strategy/defaults/hooks.yaml`:
```yaml
pre_dispatch: []
post_close: []
```

- [ ] **Step 3: Write loader test**

```go
// strategy/loader_test.go
package strategy_test

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/gh-xj/agentcli-go/strategy"
)

func TestDiscoverFromSubdir(t *testing.T) {
    tmp := t.TempDir()
    // Create .agentops/ at root
    agentopsDir := filepath.Join(tmp, ".agentops")
    os.MkdirAll(agentopsDir, 0o755)
    os.WriteFile(filepath.Join(agentopsDir, "storage.yaml"), []byte("backend: in-repo\n"), 0o644)
    os.WriteFile(filepath.Join(agentopsDir, "transitions.yaml"), []byte("categories:\n  active: [open]\ninitial: open\ntransitions: {}\n"), 0o644)

    // Create a subdirectory
    subdir := filepath.Join(tmp, "src", "pkg")
    os.MkdirAll(subdir, 0o755)

    // Load from subdirectory should find .agentops/ above
    strat, err := strategy.Discover(subdir)
    if err != nil {
        t.Fatalf("Discover failed: %v", err)
    }
    if strat.Root != tmp {
        t.Errorf("Root = %q, want %q", strat.Root, tmp)
    }
    if strat.Storage.Backend != "in-repo" {
        t.Errorf("Backend = %q, want %q", strat.Storage.Backend, "in-repo")
    }
}

func TestDiscoverMissing(t *testing.T) {
    tmp := t.TempDir()
    _, err := strategy.Discover(tmp)
    if err == nil {
        t.Fatal("expected error when .agentops/ not found")
    }
}

func TestBootstrapCreatesDefaults(t *testing.T) {
    tmp := t.TempDir()
    err := strategy.Bootstrap(tmp)
    if err != nil {
        t.Fatalf("Bootstrap failed: %v", err)
    }

    // Verify key files exist
    for _, name := range []string{"strategy.md", "schema.md", "slot.md", "storage.yaml", "transitions.yaml"} {
        path := filepath.Join(tmp, ".agentops", name)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            t.Errorf("expected %s to exist", name)
        }
    }
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./strategy/...`
Expected: FAIL

- [ ] **Step 5: Write loader.go**

```go
// strategy/loader.go
package strategy

import (
    "embed"
    "fmt"
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

//go:embed defaults/*
var defaultsFS embed.FS

// Discover walks up from startDir looking for .agentops/ and loads the strategy.
func Discover(startDir string) (*Strategy, error) {
    root, err := findRoot(startDir)
    if err != nil {
        return nil, err
    }
    return load(root)
}

// Bootstrap creates .agentops/ with default files. Idempotent.
func Bootstrap(projectDir string) error {
    agentopsDir := filepath.Join(projectDir, ".agentops")
    if err := os.MkdirAll(agentopsDir, 0o755); err != nil {
        return fmt.Errorf("create .agentops/: %w", err)
    }

    entries, err := defaultsFS.ReadDir("defaults")
    if err != nil {
        return fmt.Errorf("read embedded defaults: %w", err)
    }

    for _, entry := range entries {
        target := filepath.Join(agentopsDir, entry.Name())
        if _, err := os.Stat(target); err == nil {
            continue // don't overwrite existing
        }
        data, err := defaultsFS.ReadFile("defaults/" + entry.Name())
        if err != nil {
            return fmt.Errorf("read default %s: %w", entry.Name(), err)
        }
        if err := os.WriteFile(target, data, 0o644); err != nil {
            return fmt.Errorf("write %s: %w", entry.Name(), err)
        }
    }
    return nil
}

func findRoot(startDir string) (string, error) {
    dir, err := filepath.Abs(startDir)
    if err != nil {
        return "", err
    }
    for {
        if info, err := os.Stat(filepath.Join(dir, ".agentops")); err == nil && info.IsDir() {
            return dir, nil
        }
        parent := filepath.Dir(dir)
        if parent == dir {
            return "", fmt.Errorf("no .agentops/ found (searched up from %s)", startDir)
        }
        dir = parent
    }
}

func load(root string) (*Strategy, error) {
    agentopsDir := filepath.Join(root, ".agentops")
    s := &Strategy{Root: root}

    // Load storage.yaml
    if err := loadYAML(filepath.Join(agentopsDir, "storage.yaml"), &s.Storage); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("storage.yaml: %w", err)
    }

    // Load transitions.yaml
    if err := loadYAML(filepath.Join(agentopsDir, "transitions.yaml"), &s.Transitions); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("transitions.yaml: %w", err)
    }

    // Load risk.yaml (unstructured)
    if err := loadYAML(filepath.Join(agentopsDir, "risk.yaml"), &s.Risk); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("risk.yaml: %w", err)
    }

    // Load routing.yaml (unstructured)
    if err := loadYAML(filepath.Join(agentopsDir, "routing.yaml"), &s.Routing); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("routing.yaml: %w", err)
    }

    // Load budget.yaml (unstructured)
    if err := loadYAML(filepath.Join(agentopsDir, "budget.yaml"), &s.Budget); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("budget.yaml: %w", err)
    }

    // Load hooks.yaml
    if err := loadYAML(filepath.Join(agentopsDir, "hooks.yaml"), &s.Hooks); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("hooks.yaml: %w", err)
    }

    // Load schema.md (raw)
    if data, err := os.ReadFile(filepath.Join(agentopsDir, "schema.md")); err == nil {
        s.SchemaTemplate = string(data)
    }

    return s, nil
}

func loadYAML(path string, target any) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(data, target)
}
```

- [ ] **Step 6: Run tests**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./strategy/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add strategy/
git commit -m "feat: add strategy package for .agentops/ config loading"
```

---

## Chunk 2: Resource Implementations

Tasks 4, 5, and 6 are **independent** and can run in **parallel subagents**.

### Task 4: Case Resource

**Files:**
- Create: `resource/case/frontmatter.go`
- Create: `resource/case/frontmatter_test.go`
- Create: `resource/case/transitions.go`
- Create: `resource/case/transitions_test.go`
- Create: `resource/case/case.go`
- Create: `resource/case/case_test.go`

**Source reference:** Read casectl source at `/Users/xj/.claude/skills/case-dispatcher/cli/internal/operator/caserecord.go` and `/Users/xj/.claude/skills/case-dispatcher/cli/internal/operator/config.go` for logic to port.

- [ ] **Step 1: Write frontmatter test**

Test parsing YAML frontmatter from case.md content and rendering it back.

```go
// resource/case/frontmatter_test.go
package caseresource_test

import (
    "testing"

    caseresource "github.com/gh-xj/agentcli-go/resource/case"
)

func TestParseFrontmatter(t *testing.T) {
    content := "---\ntype: pr\nstatus: open\nclaimed_by: none\ncreated: \"2026-03-21\"\n---\n# Title\n\nBody"
    fm, body, err := caseresource.ParseFrontmatter(content)
    if err != nil {
        t.Fatalf("ParseFrontmatter: %v", err)
    }
    if fm.Type != "pr" { t.Errorf("Type = %q", fm.Type) }
    if fm.Status != "open" { t.Errorf("Status = %q", fm.Status) }
    if fm.ClaimedBy != "none" { t.Errorf("ClaimedBy = %q", fm.ClaimedBy) }
    if body != "# Title\n\nBody" { t.Errorf("body = %q", body) }
}

func TestRenderFrontmatter(t *testing.T) {
    fm := caseresource.Frontmatter{Type: "pr", Status: "open", ClaimedBy: "none", Created: "2026-03-21"}
    result := caseresource.RenderFrontmatter(fm)
    if result[:3] != "---" { t.Errorf("missing opening ---") }
}
```

- [ ] **Step 2: Implement frontmatter.go**

Port `parseCaseFrontmatter` and `renderCaseFrontmatter` from casectl `config.go`. YAML frontmatter only — no `## Metadata` fallback.

```go
// resource/case/frontmatter.go
package caseresource

import (
    "fmt"
    "strings"

    "gopkg.in/yaml.v3"
)

type Frontmatter struct {
    Type      string `yaml:"type"`
    Status    string `yaml:"status"`
    ClaimedBy string `yaml:"claimed_by"`
    Created   string `yaml:"created"`
}

func ParseFrontmatter(content string) (Frontmatter, string, error) {
    var fm Frontmatter
    if !strings.HasPrefix(content, "---\n") {
        return fm, content, fmt.Errorf("no YAML frontmatter found")
    }
    end := strings.Index(content[4:], "\n---")
    if end == -1 {
        return fm, content, fmt.Errorf("unclosed YAML frontmatter")
    }
    yamlBlock := content[4 : 4+end]
    body := strings.TrimPrefix(content[4+end+4:], "\n")
    if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
        return fm, body, fmt.Errorf("parse frontmatter: %w", err)
    }
    return fm, body, nil
}

func RenderFrontmatter(fm Frontmatter) string {
    data, _ := yaml.Marshal(fm)
    return "---\n" + string(data) + "---\n"
}
```

- [ ] **Step 3: Run frontmatter tests**

Run: `go test ./resource/case/... -run TestParseFrontmatter`
Expected: PASS

- [ ] **Step 4: Write transitions test**

```go
// resource/case/transitions_test.go
package caseresource_test

import (
    "testing"

    caseresource "github.com/gh-xj/agentcli-go/resource/case"
    "github.com/gh-xj/agentcli-go/strategy"
)

func testTransitions() strategy.TransitionsConfig {
    return strategy.TransitionsConfig{
        Categories: map[string][]string{
            "active":    {"open", "in_progress", "blocked"},
            "completed": {"resolved", "closed_no_action"},
        },
        Initial: "open",
        Transitions: map[string]strategy.TransitionDef{
            "start":   {From: "open", To: "in_progress"},
            "resolve": {From: []any{"in_progress", "blocked"}, To: "resolved"},
        },
    }
}

func TestValidTransition(t *testing.T) {
    sm := caseresource.NewStateMachine(testTransitions())
    newStatus, err := sm.Apply("open", "start")
    if err != nil { t.Fatalf("Apply: %v", err) }
    if newStatus != "in_progress" { t.Errorf("got %q, want in_progress", newStatus) }
}

func TestInvalidTransition(t *testing.T) {
    sm := caseresource.NewStateMachine(testTransitions())
    _, err := sm.Apply("resolved", "start")
    if err == nil { t.Fatal("expected error for invalid transition") }
}

func TestUnknownAction(t *testing.T) {
    sm := caseresource.NewStateMachine(testTransitions())
    _, err := sm.Apply("open", "nonexistent")
    if err == nil { t.Fatal("expected error for unknown action") }
}
```

- [ ] **Step 5: Implement transitions.go**

```go
// resource/case/transitions.go
package caseresource

import (
    "fmt"

    "github.com/gh-xj/agentcli-go/strategy"
)

type StateMachine struct {
    config strategy.TransitionsConfig
}

func NewStateMachine(config strategy.TransitionsConfig) *StateMachine {
    return &StateMachine{config: config}
}

func (sm *StateMachine) Initial() string {
    return sm.config.Initial
}

func (sm *StateMachine) Apply(currentStatus, action string) (string, error) {
    def, ok := sm.config.Transitions[action]
    if !ok {
        return "", fmt.Errorf("unknown action %q", action)
    }
    fromStates := def.FromStates()
    for _, s := range fromStates {
        if s == currentStatus {
            return def.To, nil
        }
    }
    return "", fmt.Errorf("action %q not valid from status %q (valid from: %v)", action, currentStatus, fromStates)
}

func (sm *StateMachine) AllStatuses() []string {
    seen := make(map[string]bool)
    for _, states := range sm.config.Categories {
        for _, s := range states {
            seen[s] = true
        }
    }
    result := make([]string, 0, len(seen))
    for s := range seen {
        result = append(result, s)
    }
    return result
}
```

- [ ] **Step 6: Run transitions tests**

Run: `go test ./resource/case/... -run TestTransition`
Expected: PASS

- [ ] **Step 7: Write case resource test**

Test Create (generates CASE-YYYYMMDD-slug directory), List (returns records), Get (returns single record).

```go
// resource/case/case_test.go
package caseresource_test

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    agentcli "github.com/gh-xj/agentcli-go"
    caseresource "github.com/gh-xj/agentcli-go/resource/case"
    "github.com/gh-xj/agentcli-go/dal"
    "github.com/gh-xj/agentcli-go/resource"
    "github.com/gh-xj/agentcli-go/strategy"
)

func setupTestProject(t *testing.T) (string, *strategy.Strategy) {
    t.Helper()
    tmp := t.TempDir()
    // Create .agentops/ with minimal config
    strategy.Bootstrap(tmp)
    // Use in-repo storage for simplicity
    os.WriteFile(filepath.Join(tmp, ".agentops", "storage.yaml"), []byte("backend: in-repo\n"), 0o644)
    strat, err := strategy.Discover(tmp)
    if err != nil { t.Fatal(err) }
    return tmp, strat
}

func TestCaseCreate(t *testing.T) {
    tmp, strat := setupTestProject(t)
    fs := dal.NewFileSystem()
    exec := dal.NewExecutor()
    res := caseresource.New(fs, exec, strat)

    ctx := agentcli.NewAppContext(nil)
    record, err := res.Create(ctx, "fix-login", nil)
    if err != nil { t.Fatalf("Create: %v", err) }

    today := time.Now().Format("20060102")
    expectedPrefix := "CASE-" + today + "-fix-login"
    if record.ID != expectedPrefix {
        t.Errorf("ID = %q, want prefix %q", record.ID, expectedPrefix)
    }
    if record.Kind != "case" { t.Errorf("Kind = %q", record.Kind) }

    // Verify file exists
    caseMD := filepath.Join(tmp, "cases", record.ID, "case.md")
    if _, err := os.Stat(caseMD); os.IsNotExist(err) {
        t.Errorf("case.md not created at %s", caseMD)
    }
}

func TestCaseList(t *testing.T) {
    _, strat := setupTestProject(t)
    fs := dal.NewFileSystem()
    exec := dal.NewExecutor()
    res := caseresource.New(fs, exec, strat)

    ctx := agentcli.NewAppContext(nil)
    res.Create(ctx, "first", nil)
    res.Create(ctx, "second", nil)

    records, err := res.List(ctx, nil)
    if err != nil { t.Fatalf("List: %v", err) }
    if len(records) != 2 { t.Errorf("got %d records, want 2", len(records)) }
}
```

- [ ] **Step 8: Implement case.go**

Constructor and struct skeleton:

```go
// resource/case/case.go
package caseresource

type CaseResource struct {
    fs   dal.FileSystem
    exec dal.Executor
    strat *strategy.Strategy
    sm   *StateMachine
}

func New(fs dal.FileSystem, exec dal.Executor, strat *strategy.Strategy) *CaseResource {
    var sm *StateMachine
    if strat != nil {
        sm = NewStateMachine(strat.Transitions)
    }
    return &CaseResource{fs: fs, exec: exec, strat: strat, sm: sm}
}

// casesDir resolves where cases live based on strategy.
func (r *CaseResource) casesDir() string {
    if r.strat == nil { return "" }
    if r.strat.Storage.Backend == "in-repo" {
        return filepath.Join(r.strat.Root, "cases")
    }
    // separate-repo: adjacent <project>-cases/
    repoPath := r.strat.Storage.CaseRepoPath
    if repoPath == "" {
        repoPath = "../" + filepath.Base(r.strat.Root) + "-cases"
    }
    return filepath.Join(r.strat.Root, repoPath, "cases")
}
```

Port logic from casectl `internal/operator/caserecord.go`. Key changes:
- Use `dal.FileSystem` and `dal.Executor` instead of raw `os` calls
- Read storage backend from `strategy.Strategy` instead of parsing `.casectl/`
- Return `resource.Record` instead of `CaseInfo`
- Implement `Resource`, `Validator`, and `Transitioner` interfaces
- Directory naming: `CASE-YYYYMMDD-slug` with collision suffix

This is the largest file. Port these functions from casectl:
- `CreateCase` → `(*CaseResource).Create`
- `ListCases` → `(*CaseResource).List`
- `FindCaseInSubdirs` → private helper
- `ValidateCase` → `(*CaseResource).Validate`
- `MoveCaseToGroup` → called internally by `Transition`
- `ExpandStatusFilter` → used by `List` filter handling
- `ValidateSlug` → called by `Create`

- [ ] **Step 9: Run case tests**

Run: `go test ./resource/case/... -v`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add resource/case/
git commit -m "feat: add case resource with frontmatter, transitions, CRUD"
```

---

### Task 5: Slot Resource

**Files:**
- Create: `resource/slot/worktree.go`
- Create: `resource/slot/slot.go`
- Create: `resource/slot/slot_test.go`

**Source reference:** Read casectl source at `/Users/xj/.claude/skills/case-dispatcher/cli/internal/operator/slot.go` for logic to port.

- [ ] **Step 1: Write slot test**

Test Create (creates git worktree), List (finds worktrees), Get (returns single), Delete (removes worktree), Sync (rebases).

```go
// resource/slot/slot_test.go
package slotresource_test

// Tests require a real git repo in t.TempDir() initialized with:
//   git init && git commit --allow-empty -m "init"
// Test Create, List, Get, Delete against real worktrees.
// Test Sync requires a remote (can skip in unit tests).
// Test name validation (regex ^[a-z][a-z0-9-]*$).
```

Write full tests following the casectl `test/e2e/slot_test.go` patterns but using the Resource interface.

- [ ] **Step 2: Implement worktree.go**

Port git worktree operations from casectl `slot.go`:
- `createWorktree(exec, projectDir, name)` → runs `git worktree add`
- `listWorktrees(exec, projectDir)` → parses `git worktree list --porcelain`
- `removeWorktree(exec, projectDir, name)` → checks dirty state, runs `git worktree remove`
- `syncWorktree(exec, projectDir, name)` → runs `git rebase origin/main`

Use `dal.Executor` for all git commands.

- [ ] **Step 3: Implement slot.go**

Implement `Resource`, `Deleter`, and `Syncer` interfaces. Port from casectl:
- `ValidateSlotName` → called by Create
- `CreateSlot` → `(*SlotResource).Create` — creates worktree + .slot marker
- `ListSlots` → `(*SlotResource).List` — scans git worktrees
- New: `(*SlotResource).Get` — find specific slot by name
- `RemoveSlot` → `(*SlotResource).Delete` — safety check + remove
- `SyncSlot` → `(*SlotResource).Sync` — rebase on main

- [ ] **Step 4: Run tests**

Run: `go test ./resource/slot/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add resource/slot/
git commit -m "feat: add slot resource with git worktree operations"
```

---

### Task 6: Project Resource (Scaffold)

**Files:**
- Create: `resource/project/project.go`
- Create: `resource/project/templates.go`
- Create: `resource/project/project_test.go`

**Source reference:** Read existing scaffold logic at `/Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go/service/scaffold.go` and `/Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go/service/templates.go`.

- [ ] **Step 1: Write project test**

Test that Create scaffolds a Go CLI project with expected file structure.

- [ ] **Step 2: Move templates.go**

Copy `service/templates.go` to `resource/project/templates.go`. Update package name. These are the embedded Go templates for scaffold generation.

- [ ] **Step 3: Implement project.go**

Port `service.ScaffoldService.New()` into `(*ProjectResource).Create()`. The resource only implements `Create` — List and Get return empty/not-found. Constructor accepts `dal.FileSystem` and `dal.Executor`.

Key: the `slug` parameter to Create is the project name. The opts map accepts `module`, `mode` (minimal/lean/full).

- [ ] **Step 4: Run tests**

Run: `go test ./resource/project/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add resource/project/
git commit -m "feat: add project resource (scaffold) wrapping existing templates"
```

---

## Chunk 3: CLI Layer

### Task 7: cobrax v2 — Resource Command Generation + Render Pipeline

**Files:**
- Modify: `cobrax/cobrax.go`
- Create: `cobrax/resource_commands.go`
- Create: `cobrax/resource_commands_test.go`
- Create: `cobrax/render.go`
- Create: `cobrax/render_test.go`

**Depends on:** Task 2 (resource package)

- [ ] **Step 1: Write render test**

Test table rendering, JSON rendering with field selection, TSV rendering, jq filtering.

- [ ] **Step 2: Implement render.go**

```go
// cobrax/render.go
package cobrax

// OutputMode determines how records are rendered.
// RenderRecords handles table (TTY), TSV (pipe), JSON (--json), JQ (--jq).
// Uses golang.org/x/term for isatty detection.
// Uses itchyny/gojq for jq filtering.
// JSON envelope: {"ok": true, "kind": "...", "data": [...], "warnings": [...]}
```

- [ ] **Step 3: Run render tests**

Run: `go test ./cobrax/... -run TestRender`
Expected: PASS

- [ ] **Step 4: Write resource_commands test**

Test that GenerateResourceCommands creates correct Cobra commands from a mock registry.

```go
// cobrax/resource_commands_test.go
// Register a mock resource with Schema, verify:
// - "mock create <slug>" command exists
// - "mock list" command exists
// - "mock get <id>" command exists
// Register a mock that also implements Deleter, verify "mock remove <id>" exists.
// Register a mock that does NOT implement Deleter, verify "mock remove" does NOT exist.
```

- [ ] **Step 5: Implement resource_commands.go**

```go
// cobrax/resource_commands.go
package cobrax

// GenerateResourceCommands walks registry, creates noun-verb Cobra commands.
// For each resource: create, list, get.
// For Validator: validate.
// For Deleter: remove.
// For Syncer: sync.
// For Transitioner: transition.
// Each command: parse args, call resource method, render output via render.go.
```

- [ ] **Step 6: Update cobrax.go**

Add `BuildRoot(spec RootSpec, reg *resource.Registry) *cobra.Command` that:
1. Creates root command with global flags (--json, --jq, --verbose, --no-color, --dir)
2. Calls `GenerateResourceCommands(reg, root)`
3. Returns the root command

Keep existing `Execute(spec RootSpec, args []string) int` unchanged (used by examples). Add new `ExecuteRoot(root *cobra.Command, args []string) int` for the agentops composition root.

- [ ] **Step 7: Run all cobrax tests**

Run: `go test ./cobrax/... -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add cobrax/
git commit -m "feat: cobrax v2 with resource command generation and render pipeline"
```

---

### Task 8: Standalone Commands + Composition Root

**Files:**
- Create: `cmd/agentops/main.go`
- Create: `cmd/agentops/init.go`
- Create: `cmd/agentops/doctor.go`
- Create: `cmd/agentops/dispatch.go`
- Move: `cmd/agentcli/loop.go` → `cmd/agentops/loop.go`
- Move: `cmd/agentcli/loop_core.go` → `cmd/agentops/loop_core.go`
- Move: `cmd/agentcli/loop_server.go` → `cmd/agentops/loop_server.go`
- Move: `cmd/agentcli/loop_server_core.go` → `cmd/agentops/loop_server_core.go`
- Move: `cmd/agentcli/migrate.go` → `cmd/agentops/migrate.go` (script migration, NOT casectl migration)

**Depends on:** Tasks 4-7

- [ ] **Step 1: Create cmd/agentops/main.go**

Composition root:

```go
package main

import (
    "context"
    "os"

    agentcli "github.com/gh-xj/agentcli-go"
    "github.com/gh-xj/agentcli-go/cobrax"
    "github.com/gh-xj/agentcli-go/dal"
    "github.com/gh-xj/agentcli-go/resource"
    caseresource "github.com/gh-xj/agentcli-go/resource/case"
    projectresource "github.com/gh-xj/agentcli-go/resource/project"
    slotresource "github.com/gh-xj/agentcli-go/resource/slot"
    "github.com/gh-xj/agentcli-go/strategy"
)

var appMeta = agentcli.AppMeta{
    Name:    "agentops",
    Version: "dev",
    Commit:  "none",
    Date:    "unknown",
}

func main() {
    ctx := agentcli.NewAppContext(context.Background())
    fs := dal.NewFileSystem()
    exec := dal.NewExecutor()

    // Strategy loading is optional (some commands like "new" don't need it).
    // Resources must handle nil strategy gracefully.
    strat, _ := strategy.Discover(".")

    reg := resource.NewRegistry()
    reg.Register(caseresource.New(fs, exec, strat))
    reg.Register(slotresource.New(fs, exec))
    reg.Register(projectresource.New(fs, exec))

    root := cobrax.BuildRoot(cobrax.RootSpec{
        Use:  "agentops",
        Meta: appMeta,
    }, reg)

    root.AddCommand(newInitCmd(ctx, fs))
    root.AddCommand(newDoctorCmd(ctx, reg))
    root.AddCommand(newNewCmd(ctx, reg))       // alias for "project create"
    root.AddCommand(newAddCommandCmd(ctx, fs, exec))
    root.AddCommand(newDispatchCmd(ctx, reg, strat))
    root.AddCommand(newVersionCmd())
    // Loop commands migrated from cmd/agentcli
    root.AddCommand(newLoopCmd(ctx))

    os.Exit(cobrax.ExecuteRoot(root, os.Args[1:]))
}
```

- [ ] **Step 2: Create init.go**

Port from casectl `cmd/init.go`. Calls `strategy.Bootstrap(dir)` + optionally initializes case repo.

- [ ] **Step 3: Create doctor.go**

Standalone command that iterates `reg.All()`, calls `Validate` on each resource that implements `Validator`, aggregates into a single `DoctorReport`. Also validates strategy files exist.

- [ ] **Step 4: Create dispatch.go**

Stub the 9-phase dispatch lifecycle. Phases 1-5 implemented (detect slot, load strategy, classify, assess risk, select workers). Phases 6-9 logged as TODO (worker execution is future work).

- [ ] **Step 5: Migrate loop commands**

Copy from `cmd/agentcli/` to `cmd/agentops/`: `loop.go`, `loop_core.go`, `loop_server.go`, `loop_server_core.go`, `migrate.go`. Update package names and imports. The `migrate.go` here is the script-to-Go migration tool (not casectl config migration).

- [ ] **Step 6: Create version command**

```go
func newVersionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Print version information",
        Run: func(cmd *cobra.Command, args []string) {
            fmt.Printf("agentops %s (%s, %s)\n", appMeta.Version, appMeta.Commit, appMeta.Date)
        },
    }
}
```

- [ ] **Step 7: Build and smoke test**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go build ./cmd/agentops/ && ./agentops version`
Expected: `agentops dev (none, unknown)`

Run: `./agentops --help`
Expected: Shows case, slot, project subcommands + standalone commands

- [ ] **Step 8: Commit**

```bash
git add cmd/agentops/
git commit -m "feat: add agentops binary with composition root and standalone commands"
```

---

## Chunk 4: Integration, Cleanup, Polish

### Task 9: E2E Tests

**Files:**
- Create: `test/e2e/agentops_test.go`

**Depends on:** Task 8

- [ ] **Step 1: Write E2E tests**

Port patterns from casectl `test/e2e/`. Each test:
1. Creates a temp dir with `git init`
2. Runs `agentops init`
3. Tests full workflows:
   - `init` → `case create` → `case list` → `case transition` → `case get`
   - `init` → `slot create` → `slot list` → `slot get` → `slot remove`
   - `doctor` on valid and invalid projects
   - Version output matches schema

- [ ] **Step 2: Run E2E tests**

Run: `go test ./test/e2e/... -v -timeout 120s`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add test/e2e/
git commit -m "test: add E2E tests for agentops commands"
```

---

### Task 10: Protocol Docs + Example Project

**Files:**
- Create: `protocol/*.md` (5 files from casectl)
- Create: `examples/case-tracking/.agentops/` (example project)
- Create: `schemas/case-record.schema.json`

- [ ] **Step 1: Copy protocol docs**

Copy from `/Users/xj/.claude/skills/case-dispatcher/protocol/` to `protocol/`. Update references from `.casectl/` to `.agentops/`.

- [ ] **Step 2: Create example project**

```
examples/case-tracking/
├── .agentops/
│   ├── strategy.md
│   ├── schema.md
│   ├── slot.md
│   ├── storage.yaml
│   ├── transitions.yaml
│   ├── routing.yaml
│   ├── risk.yaml
│   ├── budget.yaml
│   └── hooks.yaml
└── README.md
```

- [ ] **Step 3: Create case-record schema**

JSON schema validating the case record output format.

- [ ] **Step 4: Commit**

```bash
git add protocol/ examples/case-tracking/ schemas/case-record.schema.json
git commit -m "docs: add protocol specs, example project, case record schema"
```

---

### Task 11: Delete Old Code

**Files:**
- Delete: `cmd/agentcli/` (entire directory)
- Delete: `operator/` (entire directory)
- Delete: `service/` (entire directory)

**IMPORTANT:** This task runs LAST after all tests pass.

- [ ] **Step 1: Verify all new tests pass**

Run: `cd /Users/xj/Documents/codebase/xj-m/xj-scripts/agentcli-go && go test ./... 2>&1 | tail -20`
Expected: All PASS (old tests in operator/service may fail — that's expected since we're deleting them)

- [ ] **Step 2: Update go.mod**

Add `itchyny/gojq` dependency. Remove `google/wire` dependency (no longer needed).

Run: `go mod tidy`

- [ ] **Step 3: Delete old directories**

```bash
rm -rf cmd/agentcli/ operator/ service/
```

- [ ] **Step 4: Verify build**

Run: `go build ./cmd/agentops/`
Expected: SUCCESS

Run: `go test ./resource/... ./strategy/... ./cobrax/... ./dal/... ./test/e2e/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: delete operator/, service/, cmd/agentcli/ — replaced by resource-based architecture"
```

---

## Task Dependency Graph

```
Task 1 (exit codes) ─────────────────────────────────┐
Task 2 (resource interface) ──────────┐               │
Task 3 (strategy package) ────────────┤               │
                                      ▼               ▼
                              ┌── Task 4 (case) ──┐
                              ├── Task 5 (slot) ──┤  (parallel)
                              ├── Task 6 (project)┤
                              │                   │
                              ▼                   ▼
                        Task 7 (cobrax v2) ───────┘
                              │
                              ▼
                        Task 8 (cmd/agentops)
                              │
                              ▼
                     ┌── Task 9 (E2E tests) ──┐
                     ├── Task 10 (docs/examples)┤  (parallel)
                     │                          │
                     ▼                          ▼
                        Task 11 (cleanup)
```

**Parallelizable groups:**
- Tasks 1, 2, 3 can start immediately (1 is trivial, 2 and 3 are independent). Use worktree isolation — commits must be serialized via merge.
- Tasks 4, 5, 6 are fully independent (after 2+3). Use worktree isolation.
- Tasks 9, 10 are independent (after 8)
