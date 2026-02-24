---
name: verification-loop
description: Run deterministic agentcli loop verification with practical commands, profile-driven quality gates, and lab forensics.
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
| Canonical command surface | `agentcli loop [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [--format text|json|ndjson] [--summary path] [--no-color] [--dry-run] [--explain] [command flags]` |
| Lab command surface | `agentcli loop lab [compare|replay|run|judge|autofix] [advanced flags]` |
| Discover governance settings | `agentcli loop profiles --repo-root .` |
| Verify baseline loop health | `agentcli loop doctor --repo-root .` |
| Run quality gate | `agentcli loop quality --repo-root .` |
| Run lean gate | `agentcli loop lean --repo-root .` |
| Run behavior regression gate | `agentcli loop regression --repo-root .` |
| Update behavior baseline | `agentcli loop regression --repo-root . --write-baseline` |
| Run threshold check | `agentcli loop judge --repo-root . --threshold 9.0 --max-iterations 1` |
| Run auto-fix cycle | `agentcli loop autofix --repo-root . --threshold 9.0 --max-iterations 3` |
| Replay an iteration | `agentcli loop lab replay --repo-root . --run-id <run-id> --iter 1` |
| Compare two runs | `agentcli loop lab compare --repo-root . --run-a <run-id-a> --run-b <run-id-b> [--format md --out .docs/onboarding-loop/compare/latest.md]` |
| Advanced experiments | `agentcli loop lab run`, `... judge`, `... autofix` with `--repo-root . --mode committee --role-config <path> --max-iterations 1` |

## Profiles

Profile execution reads defaults plus repo overrides from `configs/loop-profiles.json`. Use `agentcli loop <profile>` or `agentcli loop profile <name>`.

Default profile expectations in this repo:

- `quality`: committee mode, strict threshold, verbose artifacts on.
- `lean`: lower threshold, built-in committee roles, optional for fast local checks.

Behavior regression baseline defaults to:

- `testdata/regression/loop-quality.behavior-baseline.json`

## Runbook (daily)

1. `agentcli loop doctor --repo-root .`
2. `agentcli loop quality --repo-root .`
3. Optional `agentcli loop judge --repo-root . --threshold 9.0 --max-iterations 1`
4. On failures, use `lab replay` and `lab compare` before accepting changes.

## Quick references

- [`../loop-governance/case-study.md`](../loop-governance/case-study.md)
- [`../loop-governance/SKILL.md`](../loop-governance/SKILL.md)
- [`../agents.md`](../agents.md)

## Out of scope

- This skill does not define branch strategy, PR policy, or repository-level process.
- It does not replace `loop-governance` for protocol adoption design.
