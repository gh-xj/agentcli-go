# Case Study: Loop-Gated Project Recovery

This case study shows a practical, low-cognitive-load loop workflow for project onboarding and recovery.

## Scenario

Your project has:

- Growing task surface.
- Inconsistent local onboarding behavior.
- CI failures that are hard to reproduce.

Goal: enforce one deterministic quality protocol using this repository's loop surface and keep teams moving with stable pass/fail gates.

## Step 1 — Baseline readiness check

Run:

```bash
agentops loop doctor --repo-root .
```

- If this fails, fix missing repo scaffolding before quality gates.
- If it passes, continue.

## Step 2 — Run baseline quality gate

```bash
agentops loop quality --repo-root .
```

Capture pass/fail and artifact paths from `.docs/onboarding-loop/`.

- If `judge.pass` is false, inspect the failing `judge` scores and findings.

## Step 3 — Add project profiles (one-time setup)

Create `configs/loop-profiles.json`:

```json
{
  "quality": {
    "mode": "committee",
    "role_config": "configs/skill-quality.roles.json",
    "max_iterations": 1,
    "threshold": 9.0,
    "budget": 1,
    "verbose_artifacts": true
  },
  "lean": {
    "mode": "committee",
    "max_iterations": 1,
    "threshold": 7.5,
    "budget": 1,
    "verbose_artifacts": false
  }
}
```

Verify profile loading:

```bash
agentops loop profiles --repo-root .
```

## Step 4 — Use `quality` profile for CI-friendly gates

In CI or local pre-merge checks:

```bash
agentops loop quality --repo-root .
```

This loads built-in defaults and repository override profiles, so teams use the same policy in all environments.

For low-noise local checks, run:

```bash
agentops loop lean --repo-root .
```

## Step 5 — Enforce behavior regression gate

Run:

```bash
agentops loop regression --repo-root .
```

If baseline is not initialized yet:

```bash
agentops loop regression --repo-root . --write-baseline
```

## Step 6 — Investigate failures with lab

When a quality pass fails:

- Replay one iteration:
  ```bash
  agentops loop lab replay --repo-root . --run-id <run-id> --iter 1 --threshold 9.0
  ```
- Compare two runs:
  ```bash
  agentops loop lab compare --repo-root . --run-a <run-id-a> --run-b <run-id-b> --format md --out .docs/onboarding-loop/compare/run-a-vs-run-b.md
  ```
- Run committee-style rechecks with explicit profiles:
  ```bash
  agentops loop lab judge --repo-root . --mode committee --max-iterations 2 --verbose-artifacts
  ```

## Step 7 — Optional repair cycle

For iterative fixes:

```bash
agentops loop autofix --repo-root . --threshold 9.0 --max-iterations 3
```

Re-run `doctor`, `quality`, then `judge` to confirm regression stabilization.

## Why this works as a project protocol

- One discoverable command (`loop profiles`) exposes policy intent.
- Quality gate commands are environment-agnostic.
- Failure signals are structured (`score`, `pass`, `findings`, artifacts) and replay-friendly.
- The flow stays the same for local dev and CI.
