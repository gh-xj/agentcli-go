# Committee Loop

Use committee mode to run a role-based verification experiment.

For daily low-cognitive workflow, use [`docs/loop/quickstart.md`](./quickstart.md).

## Roles

- planner: converts findings into a fix plan
- fixer: applies minimal changes
- judger: independently adds final findings before scoring

## Run

```bash
agentcli loop lab autofix \
  --mode committee \
  --role-config ./configs/committee.roles.example.json \
  --threshold 9.0 \
  --max-iterations 3 \
  --verbose-artifacts
```

Compare two runs:

```bash
agentcli loop lab compare --repo-root . --run-a 20260223-021749 --run-b 20260223-023012
```

Write a markdown compare report:

```bash
agentcli loop lab compare --repo-root . --run-a 20260223-021749 --run-b 20260223-023012 --format md --out .docs/onboarding-loop/compare/report.md
```

Replay a recorded iteration:

```bash
agentcli loop lab replay --repo-root . --run-id 20260223-021749 --iter 1
```

## External role contract

Each role may provide a command in role config. Runtime injects:

- `HARNESS_ROLE`
- `HARNESS_CONTEXT_FILE`
- `HARNESS_OUTPUT_FILE`
- `HARNESS_REPO_ROOT`

Role command should emit JSON to stdout or to `HARNESS_OUTPUT_FILE`.

Judger role is independent by default: it receives only post-fix scenario + findings context, not planner/fixer reasoning.

## Artifacts

Saved per run at:

- `.docs/onboarding-loop/runs/<run-id>/iter-XX/*`
- `.docs/onboarding-loop/runs/<run-id>/final-report.json`

This makes A/B experiments reproducible and auditable.

Reviewer summary:

- `.docs/onboarding-loop/maintainer/latest-review.md`

## Benchmark Floor

```bash
task loop:benchmark
task loop:benchmark:check
```
