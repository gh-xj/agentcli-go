# gokit

Shared Go CLI helpers for personal projects. Single flat package — no sub-packages.

## Module

`github.com/gh-xj/gokit` — import as `"github.com/gh-xj/gokit"`

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
| `cmd/gokit/main.go` | `gokit` scaffold CLI entrypoint (`new`, `add command`, `doctor`) |
| `internal/tools/schemacheck/main.go` | JSON contract validator for schema-based CI checks |
| `schemas/*.schema.json` | Versioned JSON contracts for framework outputs |

## Rules

### API Design
- All functions are exported (PascalCase) — this is a library, not a CLI
- Keep the package flat: no sub-packages, everything in package `gokit`
- Functions must be generic/reusable — no project-specific logic
- `log.Fatal` is acceptable for `RequireArg` and `CheckDependency` (CLI-oriented library)

### Dependencies
- `github.com/rs/zerolog` — structured logging
- `github.com/samber/lo` — verbose flag detection via `lo.Contains`
- No other dependencies. Keep the dep tree small.

### Adding Functions
- Only add helpers that are duplicated across 2+ CLI projects
- Follow existing patterns: short, focused, well-named
- No business logic — only generic utilities

### Versioning
- Tag releases as `v0.x.y` (pre-1.0)
- Consumers use `go get github.com/gh-xj/gokit@latest`

### Consumers
- `disk-manager` — uses aliases (`var parseArgs = gokit.ParseArgs`) for zero-churn migration
- `xj-core-maintainer` (timing-fill) — calls `gokit.ParseArgs`/`gokit.GetArg` directly
- `raycast go_scripts` — uses alias (`var runOsascript = gokit.RunOsascript`)
