# Partner Onboarding Report

- Date: 2026-02-22
- Partner: disk-manager (internal design partner)
- Repo/Use case: media import + disk workflow CLI automation
- OS: macOS
- Go version: go1.25.x
- Agent setup: terminal agent using `agentcli` scaffold workflow

## Timeline

- Start time: 2026-02-22 (local)
- First `agentcli new` success time: under 1 minute
- First `task verify` success time: about 1 minute total from start

## Metrics

- time_to_first_scaffold_success_min: 1
- time_to_first_verify_pass_min: 1
- doctor_iterations_before_green: 1
- num_contract_related_failures: 0
- num_clarification_questions: 0
- overall_onboarding_score_1_10: 9

## Friction Log

1. No functional blockers in scaffold -> doctor -> verify path.
2. Default generated command text is generic and needs domain-specific wording.
3. Partner asked for stronger install-first docs ordering (addressed in README update).

## What Worked Well

- Deterministic path was clear: `new` -> `add command` -> `doctor --json` -> `task verify`.
- `doctor --json` returned green on first pass (`ok: true`, `findings: []`).
- Generated project CI tasks were runnable without manual surgery.

## Improvement Requests

- Add richer domain templates/examples for operations-heavy CLIs.
- Add one-line installer docs for broader OSS onboarding.

## Action Items for agentcli-go

1. Keep examples runnable and domain-oriented (done for `file-sync-cli`, `http-client-cli`, `deploy-helper-cli`).
2. Continue design-partner data collection across 4+ additional repos.
3. Add Homebrew install path to reduce first-step friction.
