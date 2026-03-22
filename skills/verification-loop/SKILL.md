---
name: verification-loop
description: Run deterministic agentops loop verification with practical commands, profile-driven quality gates, and lab forensics.
version: 1.0
---

# verification-loop Skill

## In scope

- Readiness checks and baseline scoring (`doctor`, `quality`, `judge`).
- Profile-driven quality runs (`quality`).
- Advanced diagnostics (`lab replay`, `lab compare`, `lab run`, `lab judge`, `lab autofix`).

## Use this when

- You need fast, repeatable quality gates for CI or local onboarding.
- You need a structured path for fix/review/finalize with loop artifacts.
- You need to replay or compare failed iterations.

## Command map

| Intent | Command |
| --- | --- |
| Canonical command surface | `agentops loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]` |
| Lab command surface | `agentops loop lab [compare|replay|run|judge|autofix] [advanced flags]` |
| Discover governance settings | `agentops loop profiles --repo-root .` |
| Verify baseline loop health | `agentops loop doctor --repo-root .` |
| Run quality gate | `agentops loop quality --repo-root .` |
| Run lean gate | `agentops loop lean --repo-root .` |
| Run behavior regression gate | `agentops loop regression --repo-root .` |
| Update behavior baseline | `agentops loop regression --repo-root . --write-baseline` |
| Run threshold check | `agentops loop judge --repo-root . --threshold 9.0 --max-iterations 1` |
| Run auto-fix cycle | `agentops loop autofix --repo-root . --threshold 9.0 --max-iterations 3` |
| Replay an iteration | `agentops loop lab replay --repo-root . --run-id <run-id> --iter 1` |
| Compare two runs | `agentops loop lab compare --repo-root . --run-a <run-id-a> --run-b <run-id-b> [--format md --out .docs/onboarding-loop/compare/latest.md]` |
| Advanced experiments | `agentops loop lab run`, `... judge`, `... autofix` with `--repo-root . --mode committee --role-config <path> --max-iterations 1` |

Note: `--md` output is supported only for `agentops loop doctor`; quality/profile commands reject `--md`.

## Profiles

Profile execution reads defaults plus repo overrides from `configs/loop-profiles.json`. Use `agentops loop <profile>` or `agentops loop profile <name>`.

Default profile expectations in this repo:

- `quality`: committee mode, strict threshold, verbose artifacts on.
- `lean`: lower threshold, built-in committee roles, optional for fast local checks.

Behavior regression baseline defaults to:

- `testdata/regression/loop-quality.behavior-baseline.json`

## Runbook (daily)

1. `agentops loop doctor --repo-root .`
2. `agentops loop quality --repo-root .`
3. Optional `agentops loop judge --repo-root . --threshold 9.0 --max-iterations 1`
4. On failures, use `lab replay` and `lab compare` before accepting changes.

## Quick references

- [`../loop-governance/case-study.md`](../loop-governance/case-study.md)
- [`../loop-governance/SKILL.md`](../loop-governance/SKILL.md)
- [`../agents.md`](../agents.md)

## Out of scope

- This skill does not define branch strategy, PR policy, or repository-level process.
- It does not replace `loop-governance` for protocol adoption design.
