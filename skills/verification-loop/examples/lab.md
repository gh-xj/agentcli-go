# Lab Commands

Use for experiments, replay, and role-level forensics.

```bash
agentops loop quality --repo-root .
agentops loop lab run --repo-root . --mode committee --role-config ./configs/committee.roles.example.json --max-iterations 1 --verbose-artifacts
agentops loop lab run --repo-root . --mode committee --role-config ./configs/skill-quality.roles.json --max-iterations 1 --verbose-artifacts
agentops loop lab judge --repo-root . --mode committee --role-config ./configs/skill-quality.roles.json --max-iterations 1 --verbose-artifacts
agentops loop lab compare --repo-root . --run-a <run-id-a> --run-b <run-id-b> --format md --out .docs/onboarding-loop/compare/latest.md
agentops loop lab replay --repo-root . --run-id <run-id> --iter 1
```
