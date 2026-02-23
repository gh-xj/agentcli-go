# agentcli-go

[![CI](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml/badge.svg)](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gh-xj/agentcli-go)](https://github.com/gh-xj/agentcli-go/releases)
[![License](https://img.shields.io/github/license/gh-xj/agentcli-go)](./LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gh-xj/agentcli-go)](./go.mod)

Deterministic Go CLI framework for human + AI-agent teams.

Scaffold fast, verify by contract, ship with confidence.

## Project Health

- License: [Apache-2.0](./LICENSE)
- Security policy: [SECURITY.md](./SECURITY.md)
- Contribution guide: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Code of Conduct: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
- Changelog: [CHANGELOG.md](./CHANGELOG.md)
- CI workflow: [`.github/workflows/ci.yml`](./.github/workflows/ci.yml)

## Installation

### Stable Release (Recommended)

Install the scaffold CLI (Homebrew):

```bash
brew tap gh-xj/tap
brew install agentcli
```

Install the scaffold CLI (`go install`):

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.0
```

Add the framework library to your Go project:

```bash
go get github.com/gh-xj/agentcli-go@v0.2.0
```

### Development Version

If you want the latest unreleased changes from `main`:

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@main
go get github.com/gh-xj/agentcli-go@main
```

Note: `main` may include in-progress changes. Use it for early testing.

### Requirements

- Go 1.25+
- `task` (recommended for verification workflow)

## Quickstart

Generate a new CLI project:

```bash
agentcli new --module example.com/mycli mycli
```

Add a command:

```bash
agentcli add command --dir ./mycli sync-data
```

Run health checks:

```bash
agentcli doctor --dir ./mycli --json
```

Run full project verification:

```bash
cd mycli
task verify
```

## Prompt Starter (AI Onboarding)

Use this style to onboard with AI quickly, similar to `obra/superpowers`.

Copy-paste this into your coding agent:

```text
You are helping me onboard to agentcli-go.
Goal: create a deterministic Go CLI and keep it contract-compliant.

Do these steps in order and show command output summaries:
1) Check my environment and install agentcli if missing.
2) Run: agentcli new --module example.com/mycli mycli
3) Run: agentcli add command --dir ./mycli sync-data
4) Run: agentcli doctor --dir ./mycli --json
5) Run: cd mycli && task verify
6) Explain any failures and fix them.

Constraints:
- Keep output deterministic.
- Preserve schema/CI contracts.
- Do not skip verification.
```

For reusable prompt files, see [`prompts/agentcli-onboarding.prompt.md`](./prompts/agentcli-onboarding.prompt.md).

## First 5 Minutes

Copy-paste this end-to-end:

```bash
set -e
brew tap gh-xj/tap
brew install agentcli

mkdir -p /tmp/agentcli-demo && cd /tmp/agentcli-demo
agentcli new --module example.com/demo demo
agentcli add command --dir ./demo sync-data
agentcli doctor --dir ./demo --json
cd demo && task verify
```

## Documentation

- Docs home: [`docs/site/index.md`](./docs/site/index.md)
- Getting started: [`docs/site/getting-started.md`](./docs/site/getting-started.md)
- AI agent playbook: [`docs/site/ai-agent-playbook.md`](./docs/site/ai-agent-playbook.md)
- Example catalog: [`examples/README.md`](./examples/README.md)

## Why agentcli-go

Most CLI projects start fast and drift later.
`agentcli-go` is built to prevent that drift from day one.

- deterministic scaffolding for every new CLI
- machine-readable health checks (`doctor --json`)
- schema-validated smoke outputs
- CI gates with positive + negative regression checks
- Go compile-time safety for agent-generated changes

## AI-Agent Onboarding

Recommended agent workflow:

1. Run `agentcli new` for a contract-compliant baseline.
2. Use `agentcli add command` for feature growth.
3. Run `agentcli doctor --json` before/after edits.
4. Enforce `task ci` as the canonical pass/fail gate.

Why this works for agents:

- predictable layout
- deterministic output paths
- schema-backed JSON outputs
- low-ambiguity verification contract

## What You Get

### Scaffold CLI

- `agentcli new`
- `agentcli add command`
- `agentcli doctor --json`

### Runtime Modules

- `cobrax`: standardized Cobra root flags + deterministic exit-code handling
- `configx`: deterministic config merge (`Defaults < File < Env < Flags`)

### Core Helpers

- logging: `InitLogger()`
- args: `ParseArgs`, `RequireArg`, `GetArg`, `HasFlag`
- exec: `RunCommand`, `RunOsascript`, `Which`, `CheckDependency`
- fs: `FileExists`, `EnsureDir`, `GetBaseName`
- runtime contracts: `NewAppContext`, `RunLifecycle`, `NewCLIError`, `ResolveExitCode`

## Generated Project Contract

Each generated project includes:

- fixed scaffold layout
- deterministic smoke artifact: `test/smoke/version.output.json`
- schema file: `test/smoke/version.schema.json`
- canonical task gates: `fmt`, `lint`, `test`, `build`, `smoke`, `ci`, `verify`

Example structure:

```text
mycli/
  cmd/
    root.go
    <command>.go
  internal/
    app/
    config/
    io/
    tools/smokecheck/
  pkg/version/
  test/e2e/
  test/smoke/
  Taskfile.yml
  main.go
```

## Verification Model

This repository enforces JSON output contracts in CI.

- schemas: `schemas/doctor-report.schema.json`, `schemas/scaffold-version-output.schema.json`
- positive fixtures: `testdata/contracts/*.ok.json`
- negative fixtures: `testdata/contracts/*.bad-*.json`
- gates: `task schema:check`, `task schema:negative` (both part of `task ci`)

## Minimal Library Example

```go
package main

import (
	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/rs/zerolog/log"
)

func main() {
	agentcli.InitLogger()
	args := agentcli.ParseArgs([]string{"--src", "/tmp/in"})
	src := agentcli.RequireArg(args, "src", "source directory")
	log.Info().Str("src", src).Msg("ready")
}
```

## Contributing

Issues and PRs are welcome.

Before opening a PR:

1. Keep scaffold/runtime contracts deterministic.
2. Update schema fixtures when output contracts change.
3. Run `task ci`.
