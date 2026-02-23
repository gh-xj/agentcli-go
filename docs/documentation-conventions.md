# Documentation Conventions

## Purpose

Keep repository documentation changes deterministic and prevent information from landing in the wrong place.

## Canonical Destinations

- `README.md`
  - Public-facing library/project description.
  - Installation basics, examples, API overview, and high-level architecture.
  - Avoid internal agent-only workflows and internal implementation notes.

- `agents.md`
  - Agent onboarding and daily operating playbook.
- `CLAUDE.md`
  - Durable repo rules and decision records that must be remembered by future agent sessions.
- `skills/*/SKILL.md`
  - Skill-specific usage and command references.
  - Keep command signatures aligned with current CLI help output.
- `internal/...` docs/comments (`*.md`, tests, and golden fixtures)
  - Internal behavior checks and test expectations.

## Content Routing Rules

- If a change is for internal verification/harness flow, put it in `agents.md` or skill docs, not `README.md`.
- If a change is a user-facing API/library behavior change, update `README.md`.
- If a change is a durable rule that agents should remember in future sessions, update `CLAUDE.md`.
- If a change modifies skill commands or behavior, update:
  - affected `skills/*/SKILL.md`
  - `internal/tools/doccheck` expectations (if affected command signatures change)

## Hard Rules

- Do not add:
  - `agentcli loop ...` command references
  - `task ci`, `task verify`, or `.docs/...` harness path details
  - install/verification walkthroughs for the agent runtime
  - loop role/profiles comparisons or lab workflows
- to `README.md`.

- Canonical exception: the short onboarding pointer set (`README.md` -> `agents.md`) is allowed, but details stay in `agents.md` and linked skill files.

## Guardrails

- Do not add non-actionable or unverifiable claims in customer-facing docs.
- Do not duplicate onboarding flow across multiple docs; route detailed steps to one canonical file and link to it.
- Keep install + verification snippets consistent and executable.
- Any new canonical command should be discoverable from:
  - `README.md` (if user-facing) or `agents.md` (if agent-only)
  - corresponding `skills/*/SKILL.md` when applicable

## Quick Review Checklist

Before finishing docs changes, verify:

- Is this user-facing? → `README.md`
- Is this agent-operations? → `agents.md`
- Is this skill behavior? → `skills/<skill>/SKILL.md`
- Is this session memory/rules? → `CLAUDE.md`
