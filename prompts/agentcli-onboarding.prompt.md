# agentcli-go Onboarding Prompt

Use this prompt with your coding agent when starting a new CLI project.

```text
You are helping me onboard to agentcli-go.
Goal: create a deterministic Go CLI and keep it contract-compliant.

Please execute this exact flow:
1) Validate toolchain (Go, task, agentcli).
2) Create project:
   agentcli new --module example.com/mycli mycli
3) Add command:
   agentcli add command --dir ./mycli sync-data
4) Validate project contract:
   agentcli doctor --dir ./mycli --json
5) Run full verification:
   cd mycli && task verify
6) If any step fails:
   - explain root cause briefly
   - apply minimal fix
   - re-run failed verification

Rules:
- Keep generated output deterministic.
- Preserve schema/CI contracts.
- Do not claim success without verification evidence.
```
