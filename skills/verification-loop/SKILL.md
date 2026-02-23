---
name: verification-loop
description: Lean + lab verification loop skill for deterministic CLI quality gates and controlled autofix experiments.
---

# Verification Loop

## Purpose

A reusable multi-agent verification/autofix loop that works across repositories using a common judge contract.

## Interfaces

- Local CLI: `agentcli loop ...`
- API: `agentcli loop --api http://127.0.0.1:7878 ...`
- Server: `agentcli loop-server --addr 127.0.0.1:7878 --repo-root .`
- Lean loop: `agentcli loop run|judge|autofix|doctor`
- Lab committee mode: `agentcli loop lab run --mode committee --role-config <file> --verbose-artifacts`
- Compare runs: `agentcli loop lab compare --run-a <id-or-path> --run-b <id-or-path>`
- Replay iteration: `agentcli loop lab replay --run-id <id> --iter <n>`

Command reference (must match CLI help):

- `agentcli loop [run|judge|autofix|doctor]`
- `agentcli loop lab [compare|replay|run|judge|autofix]`

## Required artifacts

- `.docs/onboarding-loop/latest-summary.json`
- `.docs/onboarding-loop/maintainer/latest-review.md` (maintainer telemetry)
- per-run committee artifacts (lab + `--verbose-artifacts`): `.docs/onboarding-loop/runs/<run-id>/iter-XX/*`

## Judge contract

- Score range: `0..10`
- Pass: `score >= threshold` (default `9.0`)
- Balanced weights:
  - UX: 40%
  - Quality: 40%
  - Counter-intuitive penalties: 20%

## Adaptation points

- Scenario definitions
- Detector rules
- Fix catalog
- Branch policy
- Score threshold
- Role commands (planner/fixer/judger) with deterministic context contract
- Independent judger default (no planner/fixer reasoning context)
- Benchmark floor checks (`task loop:benchmark` + `task loop:benchmark:check`)

## Resources

- `skills/verification-loop/examples/lean.md`
- `skills/verification-loop/examples/lab.md`
- `skills/verification-loop/examples/ci.md`
- `skills/verification-loop/CHECKLIST.md`
