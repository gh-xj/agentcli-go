# Agents Guide (agentops)

This file is the quick entrypoint for coding agents working in this repository.

## Required Verification Standards

- Use `task ci` as the canonical CI contract.
- Use `task verify` for local aggregate verification.
- Preserve the loop command surface:
  - `agentops loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]`
  - `agentops loop lab [compare|replay|run|judge|autofix]`
- Do not use `agentops loop all` (not supported).

## Skill Health Checks

- Ensure these files exist and match current commands:
  - `skills/agentcli-go/SKILL.md`
  - `skills/verification-loop/SKILL.md`
  - `skills/loop-governance/SKILL.md`

- Prefer this entrypoint order for agent onboarding:
  - `skill.md`
  - `agents.md`
  - `skills/loop-governance/case-study.md`
- Keep docs/skill docs aligned with command help to satisfy `docs:check`.

## Dogfood Feedback Loop Skill

- Skill doc: `skills/dogfood-feedback-loop/SKILL.md`
- Examples: `skills/dogfood-feedback-loop/examples/*.md`
- Event contract: `schemas/dogfood-event.schema.json`
- Ledger contract: `schemas/dogfood-ledger.schema.json`

Use this flow for internal failure feedback routing:

1. Confirm GitHub auth (`gh auth status`; run `gh auth login` if needed).
2. Dry-run with event payload:
   - `task dogfood:dry-run EVENT=.docs/dogfood/event.json`
3. Publish after reviewing decision:
   - `task dogfood:publish EVENT=.docs/dogfood/event.json`
4. Re-run schema contracts when changing payload shape:
   - `task schema:check`
   - `task schema:negative`

## Install & Install Verification

- If `agentops` is not installed in the environment:
  - `go install github.com/gh-xj/agentops/cmd/agentops@v0.2.1`
  - `which agentops`
  - `agentops --version`
  - `agentops --help`

## Copy-paste onboarding script (mandatory for agents)

Use this in first contact when onboarding to this repo:

```text
I am onboarding to use this repository as an agent skill.
Project URL: https://github.com/gh-xj/agentops

1) Read in this order:
   - CLAUDE.md
   - docs/documentation-conventions.md
   - agents.md
   - skills/agentcli-go/SKILL.md
   - skills/verification-loop/SKILL.md
2) Validate installation:
   - which agentops
   - agentops --version
   - agentops --help
3) Run baseline checks:
   - task ci
   - task verify
4) For harness decisions:
   - do not run unsupported command forms (for example: `agentops loop all`)
5) Document changes with routing:
   - user-facing -> README.md
   - agent workflow -> agents.md / related skill docs
   - durable agent memory -> CLAUDE.md
```

## Harness Outputs

- Loop artifacts are under `.docs/onboarding-loop/`.
- Doctor/readiness review is written to:
  - `.docs/onboarding-loop/maintainer/latest-review.md`
- Run artifacts are persisted under:
  - `.docs/onboarding-loop/runs/<run-id>/iter-XX/`

## Reference Notes

- Durable rules and process notes are maintained in `CLAUDE.md` for this repo.

## Daily quick checks

Run in this order:

```bash
task ci
agentops loop doctor --repo-root .
agentops loop lean --repo-root .
agentops loop quality --repo-root .
agentops loop regression --repo-root .
```

For optional auto-fix iteration:

```bash
agentops loop autofix --repo-root . --threshold 9.0 --max-iterations 3
```

## Documentation convention reminder

- Treat `README.md` as customer-facing.
- Keep agent workflow specifics here in `agents.md` (or in skill docs), and durable session rules in `CLAUDE.md`.
- Canonical routing is in [docs/documentation-conventions.md](./docs/documentation-conventions.md).

- If this is not user-facing (harness/commands/install/loop/artifacts), do not place it in `README.md`.
- Use this order when adding docs:
  - `agents.md` for agent-specific operating procedure
  - `skills/*/SKILL.md` for command semantics and install checks
  - `CLAUDE.md` for durable process guarantees
