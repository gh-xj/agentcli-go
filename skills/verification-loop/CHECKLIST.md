# Reviewer Checklist

1. Open `.docs/onboarding-loop/review/latest.md` and confirm `Pass: true` with score >= threshold.
2. Confirm findings section is `none` or accepted with explicit follow-up.
3. If loop was lab-mode, verify role scores (planner/fixer/judger) are not degraded.
4. For regressions, run `agentcli loop judge --repo-root . --max-iterations 1` before merge.
5. For unclear failures, run `agentcli loop lab run --verbose-artifacts --max-iterations 1` and replay.
