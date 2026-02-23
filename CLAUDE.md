# agentcli-go

Shared Go CLI helpers and framework modules for personal projects.

## Module

`github.com/gh-xj/agentcli-go` — import as `"github.com/gh-xj/agentcli-go"`

## Architecture

| File | Purpose |
|------|---------|
| `log.go` | `InitLogger()` — zerolog + `-v`/`--verbose` flag |
| `args.go` | `ParseArgs`, `RequireArg`, `GetArg`, `HasFlag` — `--key value` CLI parsing |
| `exec.go` | `RunCommand`, `RunOsascript`, `Which`, `CheckDependency` — command execution |
| `fs.go` | `FileExists`, `EnsureDir`, `GetBaseName` — filesystem helpers |
| `core_context.go` | `AppContext`, `NewAppContext` — shared runtime context |
| `lifecycle.go` | `Hook`, `RunLifecycle` — preflight/run/postflight orchestration |
| `errors.go` | `CLIError`, `ResolveExitCode` — typed error and exit mapping |
| `scaffold.go` | `ScaffoldNew`, `ScaffoldAddCommand`, `Doctor` — golden project scaffolding and compliance checks |
| `cmd/agentcli/main.go` | `agentcli` scaffold CLI entrypoint (`new`, `add command`, `doctor`) |
| `cobrax/cobrax.go` | Cobra runtime adapter with standardized persistent flags and exit-code mapping |
| `configx/configx.go` | Deterministic config loading with precedence (`Defaults < File < Env < Flags`) |
| `internal/tools/schemacheck/main.go` | JSON contract validator for schema-based CI checks |
| `schemas/*.schema.json` | Versioned JSON contracts for framework outputs |

## Rules

### API Design
- All functions are exported (PascalCase) — this is a library, not a CLI
- Keep the package flat: no sub-packages, everything in package `agentcli`
- Functions must be generic/reusable — no project-specific logic
- `log.Fatal` is acceptable for `RequireArg` and `CheckDependency` (CLI-oriented library)

### Dependencies
- `github.com/rs/zerolog` — structured logging
- `github.com/samber/lo` — verbose flag detection via `lo.Contains`
- `github.com/spf13/cobra` — standardized command runtime in `cobrax`
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
