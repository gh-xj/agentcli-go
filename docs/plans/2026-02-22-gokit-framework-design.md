# gokit Framework Design

Date: 2026-02-22
Status: Approved (design)
Scope: Build `gokit` into a long-term, elegant, deterministic Go CLI framework suitable for AI-agent generated CLIs.

## 1. Goals

- Make `gokit` the foundational framework for fast, deterministic Go CLIs.
- Optimize for AI-agent generation: predictable structure, stable interfaces, machine-readable outputs.
- Outperform typical Python scripting stacks on reliability, performance, and maintainability.
- Preserve long-term elegance by enforcing a strict core with optional extensions.

## 2. Non-Goals

- Supporting many project layout variants at initial launch.
- Dynamic runtime plugin loading in early phases.
- Building a generic web/server framework.

## 3. Key Decisions

- Command model: Cobra-first kernel.
- Governance model: strict core, optional extensions.
- Architecture style: layered modules (not monolith, not plugin-first from day one).

## 4. Research Summary (GitHub)

Primary references used:
- Cobra: https://github.com/spf13/cobra
- GitHub CLI architecture reference: https://github.com/cli/cli
- urfave/cli: https://github.com/urfave/cli
- Kong: https://github.com/alecthomas/kong
- koanf: https://github.com/knadh/koanf
- viper: https://github.com/spf13/viper
- carapace: https://github.com/carapace-sh/carapace
- go-task: https://github.com/go-task/task

Conclusions:
- Cobra is the safest command substrate for shared long-term CLI ecosystems.
- Layered architecture scales better than monoliths for framework evolution.
- Config and runtime contracts should be modular to avoid core bloat.
- Deterministic machine-readable output is essential for agent interoperability.

## 5. Framework Shape

Recommended module roadmap:

1. `github.com/gh-xj/gokit/core`
- Current reusable helpers plus runtime contracts.
- `AppContext`, lifecycle hooks, typed error model, shared IO contracts.

2. `github.com/gh-xj/gokit/cobrax`
- Cobra adapter with standardized root setup and persistent flags.
- Shared usage/help template and completion integration.

3. `github.com/gh-xj/gokit/configx`
- Typed config loading with deterministic precedence:
  defaults < config file < env < flags.

4. `github.com/gh-xj/gokit/scaffold`
- Project generator and compliance tooling.
- `gokit new`, `gokit add command`, `gokit doctor`.

Optional extension modules (later):
- `sshx`, `httpx`, `telemetryx`, `updaterx`.

## 6. Golden Project Layout (Strict Core)

Every generated CLI follows one canonical structure:

```text
mycli/
  cmd/
    root.go
    <command>.go
  internal/
    app/
      bootstrap.go
      lifecycle.go
      errors.go
    config/
      schema.go
      load.go
    io/
      output.go
  pkg/
    version/version.go
  test/
    e2e/
      cli_test.go
      fixtures/
  Taskfile.yml
  main.go
```

Rules:
- `cmd/` defines command tree and command wiring only.
- Business logic runs through `internal/app` lifecycle.
- Required global flags:
  `--verbose`, `--config`, `--json`, `--no-color`.
- Required hooks:
  `Preflight()`, `Run()`, `Postflight()`.
- Required local/CI gates:
  `fmt`, `lint`, `test`, `build`, `smoke`, `verify`.

## 7. Reliability and Determinism Contract

Error + exit code contract:
- `0`: success
- `2`: usage/validation failure
- `3`: dependency/preflight failure
- `4`: external runtime command failure
- `10+`: domain-specific application errors

Output contract:
- Human mode: concise, stable, readable.
- Machine mode (`--json`): schema-versioned response envelope.

Determinism requirements:
- No hidden global mutable state in command logic.
- Time/random/IO boundaries injected via interfaces from `AppContext`.
- Stable output ordering for maps/lists where applicable.
- Reproducible build metadata via ldflags (`version`, `commit`, `date`).

Testing requirements:
- Unit tests for parsing/validation/config.
- Golden output tests for human + JSON modes.
- E2E binary tests for key flows.
- Determinism suite enforced by scaffold tooling.

## 8. Agent-First Capabilities

- `--json` mandatory for every command, with schema versioning.
- `gokit doctor --json` for machine-parseable diagnostics.
- Scaffold emits predictable file organization and command skeletons.
- Compliance checks enforce layout and contract invariants.

## 9. Harness Engineering Contract (Mandatory)

Framework and generated CLIs must adopt these harness rules:

- One canonical CI contract command (default: `task ci`).
- Local aggregate verification command (default: `task verify`).
- Deterministic smoke harness with fixed fixtures and explicit reset/verify steps.
- Smoke outputs persisted as JSON and validated against versioned schemas.
- Generated artifacts follow split discipline:
  - `gen:*:update` mutates files.
  - `gen:*:check` fails on drift without mutating final working state.
- CI and local checks use `check` targets, not `update` targets.
- Failure output is concise and debuggable, with artifact paths for investigation.

Invariants intake for enforceable checks:
- Read `.claude/architecture/invariants.yaml` when present.
- Enforce only machine-checkable entries marked enforceable.
- Reject ambiguous or incomplete enforceable entries (fail-fast).

## 10. Long-Term Roadmap

Phase 0: Foundation stabilization
- Preserve existing helper API.
- Define framework spec v0 and core contracts.

Phase 1: Golden scaffold MVP
- Deliver strict template generation.
- Add verification gates and determinism checks.
- Ship `gokit doctor` compliance baseline.

Phase 2: Runtime standardization
- Deliver `cobrax` + `configx`.
- Normalize root flags, help templates, config precedence.
- Require JSON output contract across commands.

Phase 3: Extension ecosystem
- Add optional modules without weakening core guarantees.
- Define extension registration manifest and lifecycle boundaries.

Phase 4: Migration + governance
- Add `gokit migrate`.
- Formalize semver policy and deprecation windows.
- Add ADR process for contract changes.

Phase 5: Platform maturity
- Adoption/compliance dashboard across repos.
- Policy-as-code checks in CI.
- Add template variants only after strict-core maturity.

## 11. Trade-off Record

Rejected: monolith framework
- Too much coupling and dependency growth risk.

Rejected: plugin-first from day one
- Too much early complexity; weaker determinism and higher agent error rate.

Accepted: layered modules + strict core + optional extensions
- Best balance of elegance, long-term scalability, and deterministic generation.

## 12. Success Metrics

- New CLI from scaffold to production-ready baseline in under 30 minutes.
- All generated projects pass `task verify` in clean environments.
- Cross-project automation can consume all command outputs via stable `--json`.
- Upgrade path remains low-risk with explicit migration tooling and contracts.

## 13. Next Step

Proceed to implementation planning (`writing-plans`) for Phase 0-1 execution details, task breakdown, sequencing, and verification gates.
