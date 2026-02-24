[![CI](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml/badge.svg)](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gh-xj/agentcli-go)](https://github.com/gh-xj/agentcli-go/releases)
[![License](https://img.shields.io/github/license/gh-xj/agentcli-go)](./LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gh-xj/agentcli-go)](./go.mod)

**Languages**: [English](./README.md) | [中文](./README.zh-CN.md)

<p align="center">
  <img src="./assets/logo/agentcli-go-logo.svg" alt="AgentCLI - GO logo" width="760" />
</p>

Framework modules for building Go CLI projects with an agent-first workflow.

`agentcli-go` is an **agent-first CLI scaffold and runtime framework**: it provides logging, argument parsing, command execution, filesystem helpers, lifecycle flow, and verification primitives so AI agents can build reliable CLI tools consistently.

---

# AgentCLI - GO

## 2-minute value

- Generate a standards-compliant Go CLI in under a minute.
- Keep it deterministic with `task ci` and docs-aware checks.
- Onboard another AI agent with one documented sequence.

## 30-second demo

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
agentcli doctor
agentcli loop doctor --repo-root .            # optional verification trace
```

Expected result: a runnable scaffold (`main.go`, `cmd/`, `Taskfile.yml`, `test/`) plus a first quality check in less than 2 minutes.
`agentcli new` now runs `go mod tidy` automatically for standalone projects, so `go.sum` is generated on scaffold.

## Why

- Skip framework boilerplate for AI-assisted CLI creation.
- Standardized CLI patterns across generated agents/tools.
- Scaffold a compliant project in under a minute.
- Machine-readable output (`--json`) built into the scaffold from day one.

### Harness Engineering value proposition

- Deterministic project scaffolding with opinionated defaults for reproducible setup.
- Built-in quality gates designed for AI-assisted workflows and CI-safe project hygiene.
- Contract-first outputs (schemas + tooling) to make generated CLIs easier to maintain over time.
- Standardized lifecycle/error semantics so teams can onboard agents and scripts faster with fewer “first-run” surprises.
- A practical base for scalable agent/tooling workflows, while keeping the runtime surface small and reviewable.

## Who should use agentcli-go

- You want AI agents to generate, evolve, and maintain Go CLIs with predictable behavior.
- You ship internal tooling and want deterministic templates, predictable outputs, and CI-friendly checks.
- You want fewer run/retry loops from inconsistent setup and missing onboarding context.
- You value a small, reviewable runtime API over large generator-heavy ecosystems.

This project is primarily a **foundation for agent-built CLIs**, and also useful for human users who want that same rigor and consistency.

## Common use cases

- Internal ops scripts (`cron`, migration helpers, maintenance assistants).
- Data/IO helpers (`sync`, fetch, transform, report).
- Multi-step agent workflows that need deterministic command/task orchestration.
- Reusable CLI kits for internal teams needing the same scaffolding and quality gates.

## How this project fits

`agentcli-go` is the **foundation layer** for CLI scripts in this repo:
- As a **library**, it provides reusable CLI primitives.
- As a **scaffold CLI**, it generates a compliant project skeleton.
- As **harness infrastructure**, it helps AI agents produce, verify, and iterate safely.

```text
User request
   │
   ▼
AI Agent (Codex/Claude/ClawHub)
   │
   ├─ reads onboarding + skill docs
   │
   ▼
agentcli-go (library + scaffold CLI)
   │
   ├─ generates standard CLI layout
   ├─ standardized flags, logging, errors, config flow
   └─ adds harness entrypoints and docs/checks
           │
           ▼
Generated project (task/verify, schemas, docs:check, loop/quality)
           │
   ▼
Lower agent cognitive load + safer iteration
```

## How this helps discoverability

- Stable, predictable output format (`--json`) and task gates help people trust what they see.
- Strong documentation structure (`README` + `agents.md` + skill docs) reduces setup confusion.
- Ready-to-use agent entrypoint on ClawHub:
  - https://clawhub.ai/gh-xj/agentcli-go

## Copy-paste onboarding prompt (for agents)

Use this short pointer at session start:

```text
I am onboarding to use this repository as an agent skill.
Project URL: https://github.com/gh-xj/agentcli-go
Please follow the sequence in agents.md, including canonical onboarding commands, doc-routing rules, and memory/update actions.
```

## Quick adoption path (5 minutes)

1. Install the CLI:

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
```

2. Generate a standard project:

```bash
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
```

3. Run the first health checks:

```bash
go test ./...
agentcli doctor
```

4. Add a command and verify quickly:

```bash
agentcli add command --name sync --preset file-sync
go run . --help
```

If this works, your team gets a scaffolded CLI with harness-friendly structure without manual setup.

If this project saves your agent setup time, give it a star on GitHub to improve discovery for other teams:
https://github.com/gh-xj/agentcli-go

## FAQ (1-minute)

- `agentcli loop` command fails in docs?
  - Check `agents.md` and `skills/verification-loop/SKILL.md` for current command signatures.
- My generated project is missing expected files?
  - Re-run `agentcli new` with a clean directory name and check template version compatibility.
- What should I do before opening PR?
  - Run `task verify` and include output of key checks in your PR notes.

## Contributing

- Contributions are welcome. See `CONTRIBUTING.md` for review expectations, local checks, and PR workflow.

## Installation

### Library (import into your project)

```bash
go get github.com/gh-xj/agentcli-go@v0.2.3
```

### Scaffold CLI (optional, for generating new projects)

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.3
```

Naming note: repository/module name is `agentcli-go`, installed binary name is `agentcli`.

Or with Homebrew:

```bash
brew tap gh-xj/tap
brew install agentcli
```

Or download a prebuilt binary (macOS/Linux amd64+arm64):

- https://github.com/gh-xj/agentcli-go/releases/tag/v0.2.3

## Claude Code Skill

For guidance on using this library effectively in Codex/Claude workflows, see [`skills/agentcli-go/SKILL.md`](./skills/agentcli-go/SKILL.md).
For agent-specific onboarding and harness entrypoints, see [`agents.md`](./agents.md).
For a project-level quick-start skill example, see [`skill.md`](./skill.md).

## Published on ClawHub

This repo is published as an agent skill at: https://clawhub.ai/gh-xj/agentcli-go

## HN drafts

- English: `docs/hn/2026-02-22-show-hn-agentcli-go.md`
- 中文: `docs/hn/2026-02-23-show-hn-agentcli-go-zh.md`

---

## Quick Start: Agent-oriented CLI example

```go
package main

import (
    "os"

    "github.com/gh-xj/agentcli-go"
    "github.com/rs/zerolog/log"
)

func main() {
    agentcli.InitLogger()
    args := agentcli.ParseArgs(os.Args[1:])

    src := agentcli.RequireArg(args, "src", "--src path")
    dst := agentcli.GetArg(args, "dst", "/tmp/out")

    if !agentcli.FileExists(src) {
        log.Fatal().Str("src", src).Msg("source not found")
    }

    agentcli.EnsureDir(dst)
    out, err := agentcli.RunCommand("rsync", "-av", src, dst)
    if err != nil {
        log.Fatal().Err(err).Msg("sync failed")
    }
    log.Info().Msg(out)
}
```

Run with: `go run . --src ./data --dst /backup`

This style of usage is intentionally minimal so agents can reuse it as a baseline and keep project rules consistent.

---

## API Reference

| Function | Description |
|----------|-------------|
| `InitLogger()` | zerolog setup with `-v`/`--verbose` for debug output |
| `ParseArgs(args)` | Parse `--key value` flags into `map[string]string` |
| `RequireArg(args, key, usage)` | Required flag — fatal if missing |
| `GetArg(args, key, default)` | Optional flag with default |
| `HasFlag(args, key)` | Boolean flag check |
| `RunCommand(name, args...)` | Run external command, return stdout |
| `RunOsascript(script)` | Execute AppleScript (macOS) |
| `Which(bin)` | Check if binary is on PATH |
| `CheckDependency(name, installHint)` | Assert dependency exists or fatal |
| `FileExists(path)` | File/dir existence check |
| `EnsureDir(path)` | Create directory tree |
| `GetBaseName(path)` | Filename without extension |

### Runtime modules

- **`cobrax`** — Cobra adapter with standardized persistent flags (`--verbose`, `--config`, `--json`, `--no-color`) and deterministic exit code mapping
- **`configx`** — Config loading with deterministic precedence: `Defaults < File < Env < Flags`

---

## Quick Start: Scaffold a New Project

Use `agentcli new` to generate a fully-wired project with Taskfile, smoke tests, and schema contracts:

```bash
agentcli new --module github.com/me/my-tool my-tool
cd my-tool
agentcli add command --preset file-sync sync-data
agentcli doctor --json        # verify compliance
task verify                   # run full local gate
```

Recommended default for monorepos: generate without nested `go.mod`:

```bash
# run inside an existing Go module
agentcli new --dir ./tools --in-existing-module replay-cli
cd tools/replay-cli
task verify
```

Minimal mode (tiny scaffold surface):

```bash
agentcli new --minimal --module github.com/me/my-mini my-mini
```

Generated layout:

```
my-tool/
├── main.go
├── cmd/root.go
├── internal/app/{bootstrap,lifecycle,errors}.go
├── internal/config/{schema,load}.go
├── internal/io/output.go
├── internal/tools/smokecheck/main.go
├── pkg/version/version.go
├── test/e2e/cli_test.go
├── test/smoke/version.schema.json
└── Taskfile.yml
```

Command presets: `file-sync`, `http-client`, `deploy-helper`, `task-replay-emit-wrapper`, `task-replay-orchestrator`

### Cross-repo orchestration pattern

Common pattern: run `task` in an external repo with env injection.

```bash
agentcli add command --preset task-replay-orchestrator replay-orchestrate
go run . replay-orchestrate \
  --repo ../external-repo \
  --task replay:emit \
  --env IOC_ID=123 \
  --env MODE=baseline \
  --timeout 90s \
  --timeout-hook 'echo timeout for $AGENTCLI_TIMEOUT_TASK'
```

Generated wrapper contract:

- required: `--repo <path>`
- optional: `--task <name>` (default `replay:emit`)
- optional repeat: `--env KEY=VALUE`
- optional: `--timeout <duration>`
- optional: `--timeout-hook <shell command>`

---

## Examples

Runnable examples:

- [`examples/file-sync-cli`](./examples/file-sync-cli)
- [`examples/http-client-cli`](./examples/http-client-cli)
- [`examples/deploy-helper-cli`](./examples/deploy-helper-cli)

Examples index: [`examples/README.md`](./examples/README.md)

---

## Project Health

- License: [Apache-2.0](./LICENSE)
- Security policy: [SECURITY.md](./SECURITY.md)
- Contribution guide: currently not available in this repository
- Code of Conduct: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
- Changelog: [CHANGELOG.md](./CHANGELOG.md)

## Documentation Conventions

- Documentation ownership and where to file updates are defined in [docs/documentation-conventions.md](./docs/documentation-conventions.md).

## For Agent-Installed Workflows

If this project is used as an agent skill, start with [`agents.md`](./agents.md), then follow links from there.

## Optional: Advanced verification profiles

`agentcli loop` supports configurable verification profiles (for automation workflows).

Canonical shape:

- `agentcli loop [global flags] <action> [action flags]`
- Global flags must appear before `<action>`.
- Global flags: `--format text|json|ndjson`, `--summary <path>`, `--no-color`, `--dry-run`, `--explain`

Discovery:

- `agentcli loop --format json capabilities`

Behavior-only regression gate:

- `agentcli loop regression --repo-root .`
- baseline update: `agentcli loop regression --repo-root . --write-baseline`

Loop API contract:

- `loop-server` endpoint `POST /v1/loop/run` returns the same harness summary envelope used by CLI output (`schema_version`, `command`, `status`, `checks`, `failures`, `artifacts`, `data`).
- Client helper `internal/loopapi.RunSummary` returns the summary envelope directly.

Core-only compile fallback:

- If optional loop/harness packages drift and you only need scaffold/doctor flows, build with `agentcli_core` tag:
  - `go build -tags agentcli_core ./cmd/agentcli`
- In `agentcli_core` builds, `loop` and `loop-server` commands are intentionally disabled.
