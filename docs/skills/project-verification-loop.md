# Project Verification Loop (Generic Skill Design)

## Purpose

A reusable multi-agent verification/autofix loop that works across repositories using a common judge contract.

## Interfaces

- Local CLI: `agentcli loop ...`
- API: `agentcli loop --api http://127.0.0.1:7878 ...`
- Server: `agentcli loop-server --addr 127.0.0.1:7878 --repo-root .`
- Committee mode: `agentcli loop all --mode committee --role-config <file>`
- Compare runs: `agentcli loop compare --run-a <id-or-path> --run-b <id-or-path>`

## Required artifacts

- `.docs/onboarding-loop/latest-summary.json`
- `.docs/onboarding-loop/findings.json`
- timestamped markdown reports
- per-run committee artifacts: `.docs/onboarding-loop/runs/<run-id>/iter-XX/*`

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
