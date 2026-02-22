# gokit Framework Implementation Plan

Date: 2026-02-22
Input design: `docs/plans/2026-02-22-gokit-framework-design.md`
Planning scope: Phase 0 and Phase 1 (foundation + golden scaffold MVP)

## 1. Objective

Execute the approved framework direction with minimal risk:
- Preserve current `gokit` utility value.
- Introduce framework contracts without breaking existing consumers.
- Deliver strict scaffold + compliance tooling for deterministic AI-agent CLI generation.

## 2. Scope

In scope:
- Core contracts and package shaping for `core`.
- Cobra adapter baseline (`cobrax`) for root command setup.
- Scaffold generator MVP (`gokit new`, `gokit add command`, `gokit doctor`).
- Determinism and verification baseline in generated projects.

Out of scope:
- Full extension marketplace.
- Template variants.
- Advanced migration automation (`gokit migrate`) beyond basic warnings.

## 3. Milestones

M0: Baseline + compatibility (1-2 days)
- Freeze current helper behavior with tests.
- Add compatibility tests for current exported APIs.
- Create architecture docs and module boundary notes.

M1: Core contracts (2-4 days)
- Add `AppContext`, hook interfaces, typed error + exit codes.
- Keep old APIs available; mark transition path in docs.
- Add deterministic output utility primitives.

M2: Cobra adapter MVP (2-3 days)
- Add standardized root command constructor.
- Add mandatory persistent flags and usage template.
- Add command execution wrapper with error->exit mapping.

M3: Scaffold MVP (3-5 days)
- `gokit new <name>` generates golden layout.
- `gokit add command <name>` adds command skeleton + wiring.
- `gokit doctor` validates structure/contracts; JSON output supported.

M4: Verification baseline (2-3 days)
- Add generated Taskfile gates (`fmt`, `lint`, `test`, `build`, `smoke`, `verify`).
- Add golden tests for text/JSON output determinism.
- Add smoke e2e for generated sample CLI.
- Add canonical CI contract command (`task ci`) and align all pipelines.

## 4. Work Breakdown

### Track A: `core` contract implementation
1. Add new files for runtime types and lifecycle interfaces.
2. Implement typed error envelope and exit code mapping.
3. Add API docs with transition guidance.
4. Add tests for hook orchestration and error mapping.

### Track B: `cobrax` runtime adapter
1. Add dependency and root command builder.
2. Standardize flags and help behavior.
3. Wrap execution with deterministic stderr/stdout behavior.
4. Add unit tests for flag presence and exit code behavior.

### Track C: `scaffold` CLI
1. Add generator entrypoint command structure.
2. Implement template renderer for golden layout.
3. Implement `add command` transform and idempotency checks.
4. Implement `doctor` checks and `--json` output mode.
5. Add integration tests for scaffold workflows.

### Track D: Generated project quality gates
1. Produce Taskfile with required commands.
2. Add basic lint/test bootstrap.
3. Add output golden testing harness.
4. Add determinism checks for sorted output and stable exit codes.
5. Add `gen:*:update` and `gen:*:check` target discipline.
6. Add schema validation for smoke JSON outputs.
7. Add artifact path conventions for CI debugging.

### Track E: Invariant enforcement intake
1. Add optional parser/validator for `.claude/architecture/invariants.yaml`.
2. Enforce required fields for enforceable invariants:
   `id`, `statement`, `status`, `evidence`, `owner`, `last_verified`.
3. Fail-fast on ambiguous or incomplete enforceable entries.
4. Convert accepted enforceable invariants to concrete harness checks.

## 5. Sequencing and Dependencies

- Do Track A first (contract definitions), because Tracks B/C depend on API.
- Track B and Track C can proceed in parallel once Track A is stable.
- Track D starts with scaffold template skeleton, finalizes after Track C commands settle.
- Track E starts after Track D command naming stabilizes.

Dependency chain:
- A -> (B + C) -> D -> E

## 6. Verification Plan

Repository-level verification (framework repo):
- `go test ./...`
- Build scaffold binary and run smoke generation in temp directory.
- Verify `doctor` detects both pass and fail cases.

Generated-project verification:
- `task fmt`
- `task lint`
- `task test`
- `task build`
- `task smoke`
- `task ci`
- `task verify`

Acceptance gates:
- All commands pass in clean environment.
- Generated project has deterministic outputs in golden tests.
- `doctor --json` output is stable and parseable.
- Smoke JSON validates against checked-in schemas.
- `gen:*:check` catches drift without mutating repo state.

## 7. Backward Compatibility Strategy

- Preserve existing top-level helper functions during phase 0-1.
- Add compatibility tests for current API signatures and behaviors.
- Document migration guidance without forcing immediate consumer rewrites.

## 8. Risks and Mitigations

Risk: Scope inflation in scaffold features.
- Mitigation: lock Phase 1 to strict core template only.

Risk: Core contracts become too abstract early.
- Mitigation: optimize for actual scaffold/runtime use first, avoid speculative interfaces.

Risk: Determinism regressions via command output changes.
- Mitigation: golden tests + explicit output contract + `doctor` checks.

## 9. Deliverables

- `docs/plans/2026-02-22-gokit-framework-design.md` (completed)
- `docs/plans/2026-02-22-gokit-framework-implementation-plan.md` (this file)
- Phase 0-1 PR series with verification evidence

## 10. Definition of Done (Phase 1)

- New CLI generated via scaffold follows golden layout exactly.
- Generated CLI includes mandatory flags and lifecycle hooks.
- `doctor` validates compliance and supports JSON output.
- `task verify` passes end-to-end in generated sample.
- Existing consumers of current helper API continue to work unchanged.
