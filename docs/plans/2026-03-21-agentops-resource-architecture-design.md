# agentops: Resource-Based Agent Operations Toolkit

**Date:** 2026-03-21
**Status:** Draft
**Approach:** Clean Abstraction Layer (Approach 2) — aggressive, no backward compatibility

## Summary

Merge casectl's case-dispatch capabilities into agentcli-go and rename the binary to `agentops`. Introduce a Resource abstraction as the central design primitive: every manageable noun (case, slot, project) implements a uniform interface, and the CLI framework auto-generates commands, output formatting, and shell completions from resource registrations.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Binary name | `agentops` | Broader than "case", signals multi-capability |
| Config directory | `.agentops/` | Matches binary, discoverable |
| Target audience | AI-assisted devs + DevOps teams | Git-native workflows agents can also drive |
| Backward compat | None | Clean break; no `.casectl/`, no `.md` configs |
| Core abstraction | Resource interface | Uniform CRUD, auto-generated commands |
| Global config format | JSON | Matches existing `configx.Load()` (JSON-based) |
| Strategy config format | YAML (machine), Markdown (prose-only) | YAML for parseable config, MD for human strategy docs |
| DI approach | Resource registry (replaces Wire) | Simpler, registry IS the composition root |
| Go module | `github.com/gh-xj/agentcli-go` | Module name unchanged, binary renamed |

## Core Abstraction: Resource System

### Resource Interface

```go
// resource/resource.go

type Record struct {
    Kind    string         // "case", "slot", "project"
    ID      string         // unique within kind
    Fields  map[string]any // arbitrary typed fields
    RawPath string         // filesystem path (if file-backed)
}

// Filter is generic key-value. Each resource interprets its own keys.
// Case uses "status", "slot". Slot uses "name", "branch". Unknown keys ignored.
type Filter map[string]string

type ResourceSchema struct {
    Kind        string
    Fields      []FieldDef
    Statuses    []string    // valid status values (if stateful)
    CreateArgs  []ArgDef
    Description string
}

type FieldDef struct {
    Name     string
    Type     string // "string", "datetime", "enum"
    Required bool
}

type ArgDef struct {
    Name        string
    Description string
    Required    bool
}

// Core interface — every resource implements this.
// Note: Create takes a user-provided "slug" (hint). The returned Record.ID
// is the canonical identifier (e.g. "CASE-20260321-fix-login" from slug "fix-login").
type Resource interface {
    Schema() ResourceSchema
    Create(ctx *agentops.AppContext, slug string, opts map[string]string) (*Record, error)
    List(ctx *agentops.AppContext, filter Filter) ([]Record, error)
    Get(ctx *agentops.AppContext, id string) (*Record, error)
}

// Optional capabilities — commands appear only if implemented.
// DoctorReport and DoctorFinding live in the root agentops package,
// importable by both resource/ and cobrax/ without cycles.
type Validator interface {
    Validate(ctx *agentops.AppContext, id string) (*agentops.DoctorReport, error)
}
type Deleter interface {
    Delete(ctx *agentops.AppContext, id string) error
}
type Syncer interface {
    Sync(ctx *agentops.AppContext, id string) error
}
type Transitioner interface {
    Transition(ctx *agentops.AppContext, id string, action string) (*Record, error)
}
```

### Resource Registry

```go
// resource/registry.go

type Registry struct {
    resources map[string]Resource
}

func NewRegistry() *Registry
func (r *Registry) Register(res Resource)
func (r *Registry) Get(kind string) (Resource, bool)
func (r *Registry) All() []Resource // sorted by kind
```

### Planned Resources

| Kind | Create | List | Get | Validate | Delete | Sync | Transition |
|---|---|---|---|---|---|---|---|
| `case` | yes | yes | yes | yes | no | no | yes |
| `slot` | yes | yes | yes (new) | no | yes | yes | no |
| `project` | yes | no | no | no | no | no | no |

Note: `project` is a degenerate Resource (Create-only). It exists in the registry so `agentops project create` is auto-generated, but `new` serves as its primary UX alias. `slot get` is new functionality not present in casectl.

## Command Auto-Generation (cobrax v2)

### Generated Commands

`cobrax.GenerateResourceCommands(registry, rootCmd)` walks the registry and creates noun-verb commands for each resource. Commands appear only for implemented interfaces.

```
agentops case create <slug>
agentops case list [--status X] [--slot Y]
agentops case get <id>
agentops case validate <id>
agentops case transition <id> <action>   # e.g. "start", "resolve", "block"

agentops slot create <name>
agentops slot list
agentops slot get <name>
agentops slot remove <name>
agentops slot sync <name>
```

### Output Pipeline (gh-inspired)

Every generated command inherits the same output rendering:

- **TTY detected:** colored table, truncated fields
- **Pipe detected:** tab-delimited, no color, full text (no flag needed)
- **`--json [fields]`:** JSON with optional field selection from `Record.Fields`
- **`--jq <expr>`:** built-in jq filtering

JSON envelope for all resources:

```json
{
  "ok": true,
  "kind": "case",
  "data": [{ "id": "...", "status": "...", ... }],
  "warnings": []
}
```

### Global Flags (inherited by all commands)

```
--json <fields>    Machine-readable JSON with required field list (gh pattern)
--jq <expr>        Filter JSON output (implies --json with all fields)
--verbose          Debug logging
--no-color         Disable color (also respects NO_COLOR env)
--dir <path>       Override project directory
```

Note: `--json` always requires a field list (e.g. `--json id,status,created`). Use `--jq .` for full JSON output without field selection. This avoids Cobra optional-value complexity.

### Standalone Commands (not auto-generated)

```
agentops init                         # bootstrap .agentops/
agentops doctor                       # cross-resource validation
agentops new <name> [--full]          # scaffold Go CLI (alias for project create)
agentops add command <name>           # add subcommand to existing scaffolded project
agentops dispatch <case-id>           # full dispatch lifecycle
agentops loop [run|judge|fix]         # verification engine
agentops version                      # build metadata
```

`new` is a UX alias for `project create` because `agentops new my-tool` reads better than `agentops project create my-tool`.

`agentops add command <name>` is a standalone command (not resource-generated) that adds a subcommand to an existing scaffolded project. It stays as a manual Cobra command alongside `new`, `init`, and `doctor`.

## Dispatch Command

`agentops dispatch <case-id>` orchestrates the full 9-phase lifecycle:

1. **detect-slot** — read `.slot` marker
2. **load-strategy** — parse `.agentops/` YAML files
3. **classify** — determine case type via `routing.yaml`
4. **assess-risk** — compute risk level via `risk.yaml`
5. **select-workers** — map (type, risk) → worker set via `budget.yaml`
6. **execute-workers** — invoke workers, collect sidecar outputs
7. **reconcile** — merge sidecars into case record
8. **fire-hooks** — execute `hooks.yaml` lifecycle hooks
9. **commit** — single atomic git commit

Workers are invoked as subprocesses or skill references. The framework validates sidecar output contracts but doesn't care about worker internals.

Agents can still orchestrate manually (read `.agentops/`, call individual commands). `dispatch` is the batteries-included path.

### Dispatch Failure Semantics

- Phases 1-5 (detect through select-workers) are **idempotent** — re-running `dispatch` repeats them safely.
- Phase 6 (execute-workers): completed sidecar outputs are **preserved on disk**. If a worker fails, previously completed sidecars survive. Re-running `dispatch` skips workers whose sidecars already exist (unless `--force`).
- Phase 7 (reconcile): only runs if all selected workers completed. Partial reconciliation is never attempted.
- Phase 9 (commit): only runs if reconcile succeeded. No partial commits.
- `dispatch` is safe to re-run after failure — it resumes from the last incomplete phase.

## Strategy Configuration

### Global Config

`~/.config/agentops/config.json` (XDG-compliant, JSON to match existing `configx.Load()`):

```json
{
  "defaults": {
    "output": "auto",
    "color": true
  },
  "git": {
    "auto_commit": true,
    "commit_prefix": ""
  },
  "scaffold": {
    "module_prefix": "github.com/myorg",
    "mode": "full"
  }
}
```

Precedence: flags > env > project JSON > global JSON > built-in defaults. Loaded via existing `configx.Load()` unchanged.

### Project Strategy

`.agentops/` directory (bootstrapped by `agentops init`):

```
.agentops/
├── strategy.md          # prose: purpose, target repos (human-only)
├── schema.md            # case record archetype template
├── slot.md              # slot naming, paths, sync policy (prose)
├── storage.yaml         # storage backend
├── transitions.yaml     # declarative state machine
├── risk.yaml            # risk classification triggers
├── routing.yaml         # type × risk → worker selection
├── budget.yaml          # risk → worker composition
├── hooks.yaml           # lifecycle hooks
└── workers/             # worker definitions
```

Strategy discovery: walk up from cwd looking for `.agentops/` (like git finds `.git/`).

### Declarative State Machine (transitions.yaml)

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

Framework enforces: no backward transitions from `completed` to `active` unless explicitly declared. Projects customize labels and transitions freely.

### Case Record Format (YAML frontmatter only)

```yaml
---
type: pr
status: open
claimed_by: none
created: "2026-03-21"
---
# Fix login timeout

## User Intent
## Findings
## Next Action
## Close Criteria
```

No `## Metadata` section parsing. Frontmatter is the single source of structured data.

## Error Handling

### Exit Codes

```go
const (
    ExitSuccess           = 0
    ExitFailure           = 1  // generic
    ExitUsage             = 2  // bad args
    ExitStrategyMissing   = 3  // no .agentops/ found
    ExitTransitionDenied  = 4  // invalid state transition
    ExitWorkerFailed      = 5  // worker returned error
    ExitValidationFailed  = 6  // case/strategy validation failed
)
```

### Doctor Output

```
$ agentops doctor
.agentops/strategy.md       ✓
.agentops/schema.md         ✓
.agentops/transitions.yaml  ✓ 5 states, 5 transitions, no dead ends
.agentops/storage.yaml      ✓ backend: separate-repo
.agentops/routing.yaml      ✗ references unknown worker "challenge"

1 finding. Run with --json for machine-readable output.
```

## Package Layout

```
github.com/gh-xj/agentcli-go/
│
├── cmd/agentops/                    # binary entry point (renamed from cmd/agentcli)
│   └── main.go
│
├── agentops.go                      # root: AppContext, AppMeta, CLIError, ExitCodes
│   errors.go
│   lifecycle.go
│
├── resource/                        # core abstraction
│   ├── resource.go                  # Resource interface, Record, Filter, Schema
│   ├── registry.go                  # Registry
│   ├── case/                        # case resource
│   │   ├── case.go                  # Create, List, Get
│   │   ├── frontmatter.go           # YAML frontmatter parse/write
│   │   └── transitions.go           # state machine enforcement
│   ├── slot/                        # slot resource
│   │   ├── slot.go                  # Create, List, Get, Delete, Sync
│   │   └── worktree.go              # git worktree operations
│   └── project/                     # scaffold resource
│       ├── project.go               # Create (scaffold)
│       └── templates.go             # embedded Go templates
│
├── dal/                             # existing data access layer (unchanged)
│   ├── interfaces.go
│   ├── filesystem.go
│   ├── exec.go
│   └── logger.go
│
├── strategy/                        # .agentops/ loading
│   ├── loader.go                    # walk-up discovery, parse all YAML
│   ├── schema.go                    # typed structs for each config file
│   └── defaults/                    # embedded default templates
│       ├── schema.md
│       ├── storage.yaml
│       ├── transitions.yaml
│       └── ...
│
├── protocol/                        # protocol contracts (documentation)
│   ├── lifecycle.md
│   ├── record.md
│   ├── worker.md
│   ├── slot.md
│   └── hooks.md
│
├── cobrax/                          # cobra adapter (extended)
│   ├── cobrax.go                    # RootSpec, Execute, global flags
│   ├── resource_commands.go         # auto-generate noun-verb commands
│   └── render.go                    # output pipeline (table/json/jq)
│
├── configx/                         # existing config merging (unchanged)
│   └── configx.go
│
├── tools/harness/                   # existing verification harness (unchanged)
│
├── internal/
│   ├── harnessloop/                 # existing loop engine (unchanged)
│   ├── loopapi/                     # existing loop API (unchanged)
│   └── dogfood/                     # existing feedback loop (unchanged)
│
├── schemas/                         # JSON schema contracts (extended)
│   ├── doctor-report.schema.json
│   ├── case-record.schema.json      # new
│   └── ...
│
└── examples/
    ├── file-sync-cli/               # existing
    ├── http-client-cli/             # existing
    ├── deploy-helper-cli/           # existing
    └── case-tracking/               # new: example .agentops/ project
```

### Dependency Direction (strict DAG)

```
            cmd/agentops
                 │
      ┌──────────┼──────────┐
      ▼          ▼          ▼
   cobrax    strategy    internal/*
      │          │
      ▼          ▼
   resource/  resource/
      │
      ▼
     dal
      │
      ▼
  root (agentops.go)
```

- `dal/` imports only root
- `strategy/` imports root + dal (uses `dal.FileSystem` for testable file I/O)
- `resource/*` imports root + dal + strategy
- `cobrax/` imports root + resource (signature changes: `BuildRoot()` replaces `NewRoot()`, `Execute()` takes `*cobra.Command` instead of `RootSpec`)
- `cmd/` imports cobrax + strategy + resource + internal
- `internal/*` imports root + dal (never resource/)

### Composition Root

```go
// cmd/agentops/main.go
func main() {
    ctx := agentops.NewAppContext(context.Background())
    strat := strategy.Load(ctx)

    reg := resource.NewRegistry()
    reg.Register(caseresource.New(ctx, strat))
    reg.Register(slotresource.New(ctx))
    reg.Register(projectresource.New(ctx))

    root := cobrax.BuildRoot(cobrax.RootSpec{
        Use:  "agentops",
        Meta: appMeta,
    }, reg)

    root.AddCommand(initCmd(ctx))
    root.AddCommand(doctorCmd(ctx, reg))
    root.AddCommand(dispatchCmd(ctx, reg, strat))
    root.AddCommand(loopCmd(ctx))

    os.Exit(cobrax.Execute(root, os.Args[1:]))
}
```

Wire DI is replaced by the registry pattern. Resources receive dal dependencies via their constructors (not via AppContext). AppContext carries context, logger, config, IO, and meta — same as today. Dal interfaces are injected at construction time:

```go
// Resource constructors accept dal dependencies explicitly
func New(ctx *agentops.AppContext, fs dal.FileSystem, exec dal.Executor, strat *strategy.Strategy) *CaseResource

// In composition root:
fs := dal.NewFileSystem()
exec := dal.NewExecutor()
reg.Register(caseresource.New(ctx, fs, exec, strat))
reg.Register(slotresource.New(ctx, fs, exec))
reg.Register(projectresource.New(ctx, fs, exec))
```

This keeps AppContext unchanged and makes dal dependencies explicit and testable.

## What Gets Deleted

| Source | Reason |
|---|---|
| `casectl` binary | Absorbed into `agentops` |
| `.casectl/` support | `.agentops/` only |
| `## Metadata` section parser | Frontmatter-only |
| `.md` config files (storage.md, routing.md, etc.) | YAML-only |
| `cmd/agentcli/` directory | Renamed to `cmd/agentops/` |
| `operator/` package | Replaced by `resource/` |
| `service/scaffold.go` | Becomes `resource/project/` |
| `service/doctor.go` | Becomes standalone `cmd/doctor.go` |
| `service/container.go` + Wire DI | Registry replaces DI container |
| `service/wire.go` | Deleted |
| casectl `cmd/case_*.go`, `cmd/slot_*.go` | Auto-generated by cobrax |
| casectl `migrate` command | No backward compat |
| Legacy worker paths (`.casectl/workers/`) | Workers register via resource interface |

## What Stays Untouched

| Source | Reason |
|---|---|
| `dal/` | Used as-is by all resource implementations |
| `configx/` | Used as-is for global config loading |
| `tools/harness/` | Verification framework, independent |
| `internal/harnessloop/` | Loop engine, independent |
| `internal/loopapi/` | Loop HTTP API, independent |
| `internal/dogfood/` | Feedback system, independent |
| `examples/file-sync-cli/` etc. | Reference projects, independent |

## Migration Path from casectl

| casectl Source | agentops Destination |
|---|---|
| `internal/operator/case.go` | `resource/case/case.go` |
| `internal/operator/slot.go` | `resource/slot/slot.go` |
| `internal/protocol/validate.go` | `strategy/loader.go` |
| `cmd/init.go` | `cmd/agentops/init.go` |
| `cmd/doctor.go` | `cmd/agentops/doctor.go` |
| Protocol docs (`protocol/*.md`) | `protocol/*.md` (moved as-is) |
| Default templates (`defaults/`) | `strategy/defaults/` (embedded) |

## Testing Strategy

- **Resource unit tests:** each resource implementation tested via interface against mock dal
- **cobrax generation tests:** verify correct commands generated from registry
- **Strategy loading tests:** verify YAML parsing, walk-up discovery, defaults
- **Transition enforcement tests:** verify state machine rejects invalid transitions
- **E2E tests:** full command execution with temp git repos (existing pattern from casectl)
- **Schema contract tests:** all JSON outputs validated against `schemas/*.schema.json`
- **Smoke tests:** `agentops version` output matches schema

## Open Questions

1. **jq dependency:** bundle a Go jq library (like `itchyny/gojq`) or shell out to `jq`? Recommendation: bundle `gojq` for zero external deps.
2. **Worker invocation in `dispatch`:** subprocess exec vs. skill reference vs. both? Recommendation: subprocess first, skill reference as future enhancement.
3. **Shell completions:** auto-generate from ResourceSchema or manual? Recommendation: auto-generate, Cobra supports this natively.

## Review Log

**2026-03-21 — Spec review round 1:** 14 findings (3 critical, 5 important, 6 suggestions). All critical and important issues resolved:
- Fixed: `Create()` parameter renamed from `id` to `slug`, documented that `Record.ID` is canonical
- Fixed: `DoctorReport` explicitly stays in root `agentops` package
- Fixed: Global config format changed from TOML to JSON (matches existing `configx.Load()`)
- Fixed: `Filter` changed from struct to `map[string]string` (generic, resource-interprets-own-keys)
- Fixed: `Validate()` takes `id` not `path` (resource resolves path internally)
- Added: `Transitioner` interface for status transitions via `transitions.yaml`
- Added: Dispatch failure semantics (idempotent re-run, sidecar preservation)
- Added: `add command` to standalone commands
- Added: Dal dependencies injected via resource constructors (not AppContext)
- Documented: `cobrax.BuildRoot()` replaces `NewRoot()`, `Execute()` signature change
- Documented: `strategy/` uses `dal.FileSystem` for testable I/O
- Documented: `--json` requires field list (gh pattern), `--jq .` for full output
- Documented: `slot get` is new functionality, `project` is degenerate Resource
