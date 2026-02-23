# agentcli-go

[![CI](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml/badge.svg)](https://github.com/gh-xj/agentcli-go/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gh-xj/agentcli-go)](https://github.com/gh-xj/agentcli-go/releases)
[![License](https://img.shields.io/github/license/gh-xj/agentcli-go)](./LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/gh-xj/agentcli-go)](./go.mod)

Deterministic Go CLI framework for human + AI-agent teams.

`agentcli-go` helps you scaffold, verify, and evolve CLIs with explicit contracts so automation stays reliable as projects grow.

## Why agentcli-go

- Build CLIs quickly with a deterministic scaffold
- Enforce machine-readable contracts (`doctor --json`)
- Catch regressions early with schema-backed CI gates
- Keep AI-agent workflows predictable and auditable

## Why This Beats Script-Based Workflows

Compared with ad-hoc Bash/Python scripts, `agentcli-go` gives you:

- compile-time safety instead of runtime surprises
- stable command contracts instead of implicit behavior drift
- deterministic verification (`task ci`, `task verify`) instead of best-effort checks
- a repeatable project shape that agents and humans can both maintain

## Installation

### 1) Install with Go (Recommended)

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1
```

Add framework library dependency in your project:

```bash
go get github.com/gh-xj/agentcli-go@v0.2.1
```

### 2) Install with Homebrew

```bash
brew tap gh-xj/tap
brew install agentcli
```

### 3) Install Prebuilt Binary

Download from releases (macOS/Linux amd64+arm64):

- https://github.com/gh-xj/agentcli-go/releases/tag/v0.2.1

### Development Version

```bash
go install github.com/gh-xj/agentcli-go/cmd/agentcli@main
go get github.com/gh-xj/agentcli-go@main
```

## Install Verification

Check the binary is on PATH and runnable:

```bash
which agentcli
agentcli --version
agentcli --help
```

Expected result:

- `which` prints a valid path
- `--version` prints a semantic version or dev version
- `--help` exits successfully and shows command usage

## Quickstart

Create a new CLI:

```bash
agentcli new --module example.com/mycli mycli
```

Add a command:

```bash
agentcli add command --dir ./mycli --preset file-sync sync-data
```

Check contract health:

```bash
agentcli doctor --dir ./mycli --json
```

Run full verification:

```bash
cd mycli
task verify
```

## First 5 Minutes

```bash
set -e

go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1

mkdir -p /tmp/agentcli-demo && cd /tmp/agentcli-demo
agentcli new --module example.com/demo demo
agentcli add command --dir ./demo --preset file-sync sync-data
agentcli doctor --dir ./demo --json
cd demo && task verify
```

## Onboarding Benchmark

Current internal partner onboarding baseline (`docs/partners/weekly-summary.md`):

- median time to first scaffold success: 1 minute
- median time to first `task verify` pass: 1 minute
- median `doctor` iterations before green: 1

If you are significantly above this baseline, treat it as onboarding friction and open an issue with your environment details.

## AI Prompt Starter

Copy-paste into your coding agent:

```text
You are helping me onboard to agentcli-go.
Goal: create a deterministic Go CLI and keep it contract-compliant.

Do these steps in order and summarize outputs:
1) agentcli new --module example.com/mycli mycli
2) agentcli add command --dir ./mycli --preset file-sync sync-data
3) agentcli doctor --dir ./mycli --json
4) cd mycli && task verify

If anything fails, fix root cause and re-run verification.
Do not skip contract checks.
```

Optional: add a clearer command description during scaffold:

```bash
agentcli add command --dir ./mycli --description "sync local files" sync-data
```

Available presets for `agentcli add command --preset`:

- `file-sync`
- `http-client`
- `deploy-helper`

List presets from CLI:

```bash
agentcli add command --list-presets
```

Reusable prompt: [`prompts/agentcli-onboarding.prompt.md`](./prompts/agentcli-onboarding.prompt.md)

## Core Capabilities

### Scaffold CLI

- `agentcli new`
- `agentcli add command`
- `agentcli doctor --json`

### Runtime Modules

- `cobrax`: standardized Cobra runtime + deterministic exit handling
- `configx`: deterministic config layering (`Defaults < File < Env < Flags`)

### Core Helpers

- logging: `InitLogger()`
- args: `ParseArgs`, `RequireArg`, `GetArg`, `HasFlag`
- exec: `RunCommand`, `RunOsascript`, `Which`, `CheckDependency`
- fs: `FileExists`, `EnsureDir`, `GetBaseName`
- runtime contracts: `NewAppContext`, `RunLifecycle`, `NewCLIError`, `ResolveExitCode`

## Verification Contract

Generated projects include:

- deterministic smoke artifact: `test/smoke/version.output.json`
- schema file: `test/smoke/version.schema.json`
- canonical gates: `fmt`, `lint`, `test`, `build`, `smoke`, `ci`, `verify`

This repository enforces output contracts using:

- `schemas/doctor-report.schema.json`
- `schemas/scaffold-version-output.schema.json`
- positive fixtures: `testdata/contracts/*.ok.json`
- negative fixtures: `testdata/contracts/*.bad-*.json`

## Examples

Runnable examples:

- [`examples/file-sync-cli`](./examples/file-sync-cli)
- [`examples/http-client-cli`](./examples/http-client-cli)
- [`examples/deploy-helper-cli`](./examples/deploy-helper-cli)

Examples index: [`examples/README.md`](./examples/README.md)

## Documentation

Simple docs entry points:

- https://gh-xj.github.io/agentcli-go/
- [`docs/site/index.md`](./docs/site/index.md)
- [`docs/site/getting-started.md`](./docs/site/getting-started.md)
- [`docs/site/ai-agent-playbook.md`](./docs/site/ai-agent-playbook.md)

## Project Health

- License: [Apache-2.0](./LICENSE)
- Security policy: [SECURITY.md](./SECURITY.md)
- Contribution guide: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Code of Conduct: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)
- Changelog: [CHANGELOG.md](./CHANGELOG.md)

## Contributing

Before opening a PR:

1. Keep scaffold/runtime behavior deterministic.
2. Update schema fixtures when output contracts change.
3. Run `task ci`.

## Maintainer Release Flow

```bash
task release VERSION=v0.2.2
```

Requirements:

- `docs/releases/v0.2.2.md` exists
- GitHub CLI (`gh`) authenticated for `gh-xj/agentcli-go`

Optional maintainer steps:

```bash
task release:verify VERSION=v0.2.2
task release:tap VERSION=v0.2.2 TAP_PUSH=1
task release:all VERSION=v0.2.2 TAP_PUSH=1
```
