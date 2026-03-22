# agentops Onboarding Prompt

Use this prompt with your coding agent when onboarding to this repository.

```text
You are helping me onboard to agentops.
Goal: understand the repository, validate the toolchain, and keep local verification green.

Please execute this exact flow:
1) Install agentops:
   go install github.com/gh-xj/agentops/cmd/agentops@v0.2.1
2) Validate binary and toolchain:
   which agentops
   agentops --version
   agentops --help
3) Read in this order:
   CLAUDE.md
   docs/documentation-conventions.md
   agents.md
   skills/agentcli-go/SKILL.md
   skills/verification-loop/SKILL.md
4) Run repository verification:
   task ci
   task verify
5) If any step fails:
   - explain root cause briefly
   - apply minimal fix
   - re-run failed verification

Rules:
- Keep command references aligned with the live `agentops` CLI.
- Preserve schema/CI contracts.
- Do not claim success without verification evidence.
```
