# Agents Guide (agentcli-go)

This file is the quick entrypoint for coding agents working in this repository.

## Required Verification Standards

- Use `task ci` as the canonical CI contract.
- Use `task verify` for local aggregate verification.
- Preserve the loop command surface:
  - `agentcli loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]`
  - `agentcli loop lab [compare|replay|run|judge|autofix]`
- Do not use `agentcli loop all` (not supported).

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

## Install & Install Verification

- If `agentcli` is not installed in the environment:
  - `go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1`
  - `which agentcli`
  - `agentcli --version`
  - `agentcli --help`

## Copy-paste onboarding script (mandatory for agents)

Use this in first contact when onboarding to this repo:

```text
I am onboarding to use this repository as an agent skill.
Project URL: https://github.com/gh-xj/agentcli-go

1) Read in this order:
   - CLAUDE.md
   - docs/documentation-conventions.md
   - agents.md
   - skills/agentcli-go/SKILL.md
   - skills/verification-loop/SKILL.md
2) Validate installation:
   - which agentcli
   - agentcli --version
   - agentcli --help
3) Run baseline checks:
   - task ci
   - task verify
4) For harness decisions:
   - do not run unsupported command forms (for example: `agentcli loop all`)
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
agentcli loop doctor --repo-root .
agentcli loop lean --repo-root .
agentcli loop quality --repo-root .
agentcli loop regression --repo-root .
```

For optional auto-fix iteration:

```bash
agentcli loop autofix --repo-root . --threshold 9.0 --max-iterations 3
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
