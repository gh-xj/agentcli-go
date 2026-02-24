# Changelog

All notable changes to this project will be documented in this file.

## [0.2.5] - 2026-02-24

### Added
- new `agentcli migrate` command for existing shell-script repos (`scripts/*.sh`) with dual modes:
  - `--mode safe` (default, parallel workspace)
  - `--mode in-place`
- migration analysis/generation pipeline under `internal/migrate`:
  - script scan (`bash`/`sh`) with risk signal detection
  - deterministic strategy planner (`auto`, `wrapper`, `manual`)
  - artifact generator (`docs/migration/{plan,report,compatibility}` + migration skill scaffold)
- migration contract schemas and fixtures:
  - `schemas/migrate-plan.schema.json`
  - `schemas/migrate-report.schema.json`
  - `testdata/contracts/migrate-*.json`

### Changed
- root help now includes migration onboarding guidance for AI agents via `agentcli --help`
- `task schema:check` and `task schema:negative` now validate migration contracts
- docs and skill guidance updated for script migration onboarding

## [0.2.4] - 2026-02-24

### Added
- `agentcli new --minimal` scaffold mode for low-cognitive-load bootstrapping
- `task-replay-orchestrator` preset with `--timeout` and `--timeout-hook`
- release gate hardening (`task release:gate`) including `go run ./cmd/agentcli --help`
- core-only compile fallback (`-tags agentcli_core`) so core CLI builds even if optional loop/harness packages drift

### Changed
- loop CLI now calls `tools/harness/commands` directly (removed legacy adapter wrappers in `cmd/agentcli/loop.go`)
- strict static analysis cleanup (`RunLifecycle` context handling, loop profile conversion, skillquality return simplification)
- README/README.zh-CN examples and install targets now reference `v0.2.4`

### Removed
- deprecated `task-replay-emit-wrapper` command preset (consolidated on `task-replay-orchestrator`)

## [0.2.3] - 2026-02-24

### Added
- loop behavior regression snapshot engine and baseline support
- quality profile policy files (`configs/loop-profiles.json`, `configs/skill-quality.roles.json`)
- skill quality judger tool (`internal/tools/skillquality`)
- project-level skills: `agentcli-go` and `loop-governance`
- Chinese README (`README.zh-CN.md`)

### Changed
- CI now includes `regression:behavior` gate in `task ci`
- loop docs/scripts aligned to `quality`, `lean`, and `autofix` flow
- docs site simplified to a single architecture diagram

## [0.2.2] - 2026-02-24

### Added
- `agentcli new --in-existing-module` mode for monorepos (scaffold without nested `go.mod`)
- new `add command` preset: `task-replay-emit-wrapper` for cross-repo Task execution with env injection
- loop API summary contract helper `internal/loopapi.RunSummary`

### Changed
- scaffold generation now runs `go mod tidy` automatically for standalone projects to emit `go.sum`
- loop adapter modularized with handlers moved under `tools/harness/commands`
- README expanded with monorepo and cross-repo orchestration guidance

## [0.2.0] - 2026-02-22

### Added
- `agentcli` scaffold CLI (`new`, `add command`, `doctor --json`)
- deterministic schema validation gates in CI
- `cobrax` and `configx` runtime modules

### Changed
- standardized on `cobrax` scaffold runtime
- renamed project/module to `agentcli-go`
