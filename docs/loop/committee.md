# Committee Loop

Use committee mode to run a role-based verification experiment.

## Roles

- planner: converts findings into a fix plan
- fixer: applies minimal changes
- judger: independently adds final findings before scoring

## Run

```bash
agentcli loop all \
  --mode committee \
  --role-config ./configs/committee.roles.example.json \
  --threshold 9.0 \
  --max-iterations 3
```

## External role contract

Each role may provide a command in role config. Runtime injects:

- `HARNESS_ROLE`
- `HARNESS_CONTEXT_FILE`
- `HARNESS_OUTPUT_FILE`
- `HARNESS_REPO_ROOT`

Role command should emit JSON to stdout or to `HARNESS_OUTPUT_FILE`.

## Artifacts

Saved per run at:

- `.docs/onboarding-loop/runs/<run-id>/iter-XX/*`
- `.docs/onboarding-loop/runs/<run-id>/final-report.json`

This makes A/B experiments reproducible and auditable.
