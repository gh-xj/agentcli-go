# gokit

Shared Go CLI helpers for personal projects. Eliminates copy-pasting `initLogger`, `parseArgs`, `fileExists`, etc. across CLIs.

## Install

```bash
go get github.com/gh-xj/gokit@latest
```

## Usage

```go
package main

import (
    "github.com/gh-xj/gokit"
    "github.com/rs/zerolog/log"
)

func main() {
    gokit.InitLogger() // reads -v/--verbose from os.Args

    args := gokit.ParseArgs(os.Args[1:])
    src := gokit.RequireArg(args, "src", "source directory")
    dest := gokit.GetArg(args, "dest", "/tmp")

    gokit.CheckDependency("rsync", "brew install rsync")
    gokit.EnsureDir(dest)

    log.Info().Str("src", src).Str("dest", dest).Msg("ready")
}
```

## API

| Package | Function | Description |
|---------|----------|-------------|
| log | `InitLogger()` | zerolog setup with `-v`/`--verbose` support |
| args | `ParseArgs(args) map` | `--key value` parser |
| args | `RequireArg(args, key, usage) string` | Required flag or fatal |
| args | `GetArg(args, key, default) string` | Optional flag with default |
| args | `HasFlag(args, key) bool` | Boolean flag check |
| exec | `RunCommand(name, args...) (string, error)` | Run command, capture stdout |
| exec | `RunOsascript(script) string` | Execute AppleScript |
| exec | `Which(cmd) bool` | Check if command exists |
| exec | `CheckDependency(name, hint)` | Fatal if command missing |
| fs | `FileExists(path) bool` | Path existence check |
| fs | `EnsureDir(dir) error` | Create directory tree |
| fs | `GetBaseName(path) string` | Filename without extension |
| core | `NewAppContext(ctx) *AppContext` | Shared runtime context with logger/io defaults |
| core | `RunLifecycle(app, hook, run) error` | Standardized preflight/run/postflight flow |
| core | `NewCLIError(code, kind, message, cause)` | Typed CLI error with deterministic exit code |
| core | `ResolveExitCode(err) int` | Map errors to process exit codes |

## Scaffold CLI (Phase 1)

Use the scaffold CLI to generate and validate golden-layout projects:

```bash
go run ./cmd/gokit new --module example.com/mycli mycli
go run ./cmd/gokit add command --dir ./mycli sync-data
go run ./cmd/gokit doctor --dir ./mycli --json
```

Generated projects include a deterministic smoke artifact contract:
- writes `test/smoke/version.output.json`
- validates output against `test/smoke/version.schema.json`

This repo also enforces framework JSON contracts in CI:
- schemas: `schemas/doctor-report.schema.json`, `schemas/scaffold-version-output.schema.json`
- fixtures: `testdata/contracts/*.ok.json`, `testdata/contracts/*.bad-*.json`
- gates: `task schema:check` and `task schema:negative` (both included in `task ci`)
