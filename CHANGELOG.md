# Changelog

All notable changes to this project will be documented in this file.

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
