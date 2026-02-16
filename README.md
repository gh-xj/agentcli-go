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
