# Reviewer Checklist

1. Open `.docs/onboarding-loop/maintainer/latest-review.md` and confirm `Pass: true` with score >= threshold.
2. Confirm findings section is `none` or accepted with explicit follow-up.
3. If loop was lab-mode, verify role scores (planner/fixer/judger) are not degraded.
4. For skill package quality, run `agentops loop quality --repo-root .`.
5. For behavior regressions, run `agentops loop regression --repo-root .` before merge.
6. For unclear failures, run `agentops loop lab run --verbose-artifacts --max-iterations 1` and replay.
