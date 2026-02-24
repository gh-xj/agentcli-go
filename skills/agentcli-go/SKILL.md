---
name: agentcli-go
description: Use when scaffolding, maintaining, or debugging agentcli-go CLIs. Covers library API usage, scaffold workflows, and conventions that affect verification.
version: 1.0
---

# agentcli-go Skill

## In scope

- `agentcli` package usage (`ParseArgs`, `RunCommand`, `FileExists`, etc.).
- Scaffold flow (`agentcli new`, `agentcli add command`, `agentcli doctor`).
- Migration flow for existing shell scripts (`agentcli migrate`).
- Runtime conventions for `cobrax`, config precedence, and `task verify` compatibility.

## Use this when

- You need expected output for a new scaffold project.
- You need to migrate existing `scripts/*.sh` into agentcli-go structure.
- You need migration/debugging guidance for helper APIs.
- You need predictable CLI behavior before implementation decisions.

## Install quick check

- `go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.5`
- `which agentcli`
- `agentcli --help`

## Migration quick run

- `agentcli migrate --source ./scripts --mode safe --dry-run`
- `agentcli migrate --source ./scripts --mode safe --apply`
- Review `docs/migration/report.md` and `docs/migration/report.json`.

## Core references

| Topic | Source |
| --- | --- |
| API references | `README.md` |
| Args and execution helpers | `args.go`, `exec.go` |
| Cobra conventions | `cobrax/cobrax.go` |
| Scaffold templates | `scaffold.go`, `scaffold_test.go` |

## Loop integration

- `skills/loop-governance/SKILL.md` for project-level loop policy.
- `skills/verification-loop/SKILL.md` for command-level quality workflows.

## Agent onboarding

Use [`../agents.md`](../agents.md) for the local agent checklist.

## Out of scope

- This skill does not define profile policy or loop strategy.
- It does not replace project-specific verification playbooks in `skills/verification-loop/*`.
