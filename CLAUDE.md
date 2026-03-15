# agentcli-go

Shared Go CLI helpers and framework modules for personal projects.

## Module

`github.com/gh-xj/agentcli-go` — import as `"github.com/gh-xj/agentcli-go"`

## Architecture

DAG-layered: `cmd/ (handler) → service → operator → dal`, with root package as model layer.

### Root Package (Model Layer)
| File | Purpose |
|------|---------|
| `core_context.go` | `AppContext`, `IOStreams`, `AppMeta` — shared runtime contracts |
| `lifecycle.go` | `Hook` interface — preflight/postflight contract |
| `errors.go` | `CLIError`, `ExitCoder`, `ResolveExitCode` — typed error and exit mapping |

### DAL Layer (`dal/`)
| File | Purpose |
|------|---------|
| `dal/interfaces.go` | `FileSystem`, `Executor`, `Logger` interfaces + `DirEntry` |
| `dal/filesystem.go` | `FileSystemImpl` — real filesystem operations |
| `dal/exec.go` | `ExecutorImpl` — command execution, PATH lookup |
| `dal/logger.go` | `LoggerImpl` — zerolog setup |

### Operator Layer (`operator/`)
| File | Purpose |
|------|---------|
| `operator/interfaces.go` | `TemplateOperator`, `ComplianceOperator`, `ArgsOperator` |
| `operator/template_op.go` | Template rendering, go.mod parsing, parent module resolution |
| `operator/compliance_op.go` | File existence/content checks, command name validation |
| `operator/args_op.go` | `--key value` CLI argument parsing |

### Service Layer (`service/`)
| File | Purpose |
|------|---------|
| `service/container.go` | Wire DI container with `Get()` singleton |
| `service/wire.go` | Wire provider set binding all layers |
| `service/scaffold.go` | `ScaffoldService` — project generation + `AddCommand` |
| `service/doctor.go` | `DoctorService` — compliance checks with DAG validation |
| `service/lifecycle.go` | `LifecycleService` — preflight/run/postflight orchestration |
| `service/templates.go` | All scaffold template constants |

### Adapters & CLI
| File | Purpose |
|------|---------|
| `cobrax/cobrax.go` | Cobra runtime adapter with standardized persistent flags and exit-code mapping |
| `configx/configx.go` | Deterministic config loading with precedence (`Defaults < File < Env < Flags`) |
| `cmd/agentcli/main.go` | `agentcli` scaffold CLI entrypoint (`new`, `add command`, `doctor`) |
| `internal/tools/schemacheck/main.go` | JSON contract validator for schema-based CI checks |
| `schemas/*.schema.json` | Versioned JSON contracts for framework outputs |

### Dependency Direction (enforced by Go imports)
```
cmd/agentcli → service → operator → dal
                 ↓           ↓        ↓
              root (model: AppContext, Hook, CLIError)
```
- `dal` imports root only
- `operator` imports root + dal
- `service` imports root + operator + dal
- `cmd` imports service + root

### Deprecated Root Functions
Root-level functions (`ScaffoldNew`, `RunCommand`, `FileExists`, `ParseArgs`, `InitLogger`, etc.) are deprecated. Use DAG layer equivalents via `service.Get()`.

## Rules

### API Design
- Root package = shared contracts only (types, interfaces, error codes)
- New functionality goes in the appropriate DAG layer (dal, operator, or service)
- All exported types/functions use PascalCase
- Operators return errors (never `log.Fatal`)

### Dependencies
- `github.com/rs/zerolog` — structured logging
- `github.com/samber/lo` — verbose flag detection via `lo.Contains`
- `github.com/spf13/cobra` — standardized command runtime in `cobrax`
- `github.com/google/wire` — compile-time dependency injection
- Keep dependencies minimal and justified.

### Adding Functions
- Only add helpers that are duplicated across 2+ CLI projects
- Follow existing patterns: short, focused, well-named
- No business logic — only generic utilities

### Versioning
- Tag releases as `v0.x.y` (pre-1.0)

## Documentation Conventions

- Route updates according to `docs/documentation-conventions.md`.
- Avoid duplicating detailed onboarding or harness instructions in multiple top-level docs.
- Keep user-facing, agent-facing, skill-facing, and durable rules in their designated documents.

## AgentCLI Harness Learnings (2026-02-23)
- In this repo, `agentcli loop all` is not a supported command; valid actions are `run|judge|autofix|doctor|quality`, and `lab` for advanced actions.
- Use `task ci` as the canonical CI gate and `task verify` as the local aggregate verification entrypoint.
- Keep `docs:check` aligned: `internal/tools/doccheck` expects `skills/verification-loop/SKILL.md`, so this file must exist and contain current loop command signatures.
- In repo docs/scripts, prefer install/verification commands that are actually available in-repo (`go install ...`, `which agentcli`, `agentcli --version`, `agentcli --help`) and avoid references to missing external helper scripts.
