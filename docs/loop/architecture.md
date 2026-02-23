# Verification Loop Architecture

```mermaid
flowchart TD
  A[agentcli loop run|judge|autofix] --> B[harnessloop.RunLoop]
  B --> C[Default Onboarding Scenario]
  C --> D[Detect Findings]
  D --> E[Judge]
  E --> F[Write latest-summary.json]
  F --> G[Write maintainer/latest-review.md]

  H[agentcli loop lab ...] --> I[Committee Engine]
  I --> J[planner role]
  I --> K[fixer role]
  I --> L[judger role]
  J --> M[iter artifacts optional]
  K --> M
  L --> M
  M --> N[runs/<run-id>/final-report.json]

  O[agentcli loop lab compare] --> P[compare markdown/json]
  O2[agentcli loop lab replay] --> Q[replay report]
```

## Lean Path

- Commands: `run`, `judge`, `autofix`, `doctor`
- Output focus: `.docs/onboarding-loop/latest-summary.json` and `.docs/onboarding-loop/maintainer/latest-review.md`

## Lab Path

- Commands under `agentcli loop lab ...`
- Supports compare/replay/role experiments
- Per-iteration artifacts are enabled with `--verbose-artifacts`


D2 source: [docs/loop/architecture.d2](./architecture.d2)
