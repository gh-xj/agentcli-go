# AI Agent Playbook

Use this flow for deterministic agent-driven CLI development:

1. Bootstrap with `agentcli new`.
2. Add commands with `agentcli add command`.
3. Validate before and after edits: `agentcli doctor --json`.
4. Enforce quality gate: `task ci`.

If command output contracts change, update schema fixtures and tests in the same PR.

## Copy-Paste Prompt

Use this prompt pattern for fast onboarding:

```text
You are helping me onboard to agentcli-go.
Goal: create a deterministic Go CLI and keep it contract-compliant.

Do these steps in order:
1) agentcli new --module example.com/mycli mycli
2) agentcli add command --dir ./mycli sync-data
3) agentcli doctor --dir ./mycli --json
4) cd mycli && task verify

If anything fails, fix and re-run verification.
Do not skip contract checks.
```

Reusable prompt file: `prompts/agentcli-onboarding.prompt.md`
