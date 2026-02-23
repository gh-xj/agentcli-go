# Lean Commands

Use for daily checks with minimal cognitive load.

```bash
agentcli loop doctor --repo-root . --md
agentcli loop judge --repo-root . --threshold 9.0 --max-iterations 1
agentcli loop autofix --repo-root . --threshold 9.0 --max-iterations 3
```
