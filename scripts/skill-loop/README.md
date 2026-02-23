# Skill Loop Trigger

Single-entry project skill trigger for local self-evolution with human-in-the-loop PR review.

## What it does

1. Creates isolated worktree from `origin/main`.
2. Runs local loop (`agentcli loop all`) until judge threshold or max iterations.
3. Pushes dedicated branch.
4. Opens/updates PR for human review.

## Usage

```bash
BASE_BRANCH=main LOOP_BRANCH=autofix/onboarding-loop THRESHOLD=9.0 MAX_ITERATIONS=5 \
  scripts/skill-loop/run.sh
```

## Optional env vars

- `BASE_BRANCH` (default: `main`)
- `SOURCE_REF` (default: `origin/<BASE_BRANCH>`, can be `HEAD` for local dry runs)
- `LOOP_BRANCH` (default: `autofix/onboarding-loop`)
- `THRESHOLD` (default: `9.0`)
- `MAX_ITERATIONS` (default: `5`)
- `WORKTREE_DIR` (default: `/tmp/agentcli-evolve-<timestamp>`)
- `KEEP_WORKTREE=1` to keep worktree after run
- `DRY_RUN=1` to skip push/PR creation
