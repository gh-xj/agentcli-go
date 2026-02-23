# Project Self-Evolve Skill

## Expected flow

1. Human triggers project skill (`task loop:evolve`).
2. Local harness loop runs in isolated worktree.
3. Loop applies autofixes until judge score reaches threshold.
4. Branch is pushed and PR is opened/updated.
5. Human reviews and decides merge.

## Trigger

```bash
task loop:evolve
```

With overrides:

```bash
task loop:evolve BASE_BRANCH=main LOOP_BRANCH=autofix/onboarding-loop THRESHOLD=9.0 MAX_ITERATIONS=5
```

## Human-in-the-loop contract

- Loop can change code on dedicated branch only.
- No auto-merge.
- Final decision is always human PR review.
