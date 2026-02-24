# Migration Reporting

This document defines migration artifacts produced by:

```bash
agentcli migrate --source ./scripts --mode safe --apply
```

## Artifacts

- `docs/migration/plan.json`: planned strategy for each script (`auto`, `wrapper`, `manual`).
- `docs/migration/report.json`: structured migration summary for automation.
- `docs/migration/report.md`: human-readable migration summary.
- `docs/migration/compatibility.md`: shell compatibility and risk notes.

## KPI Fields for Onboarding Conversion

Track these fields in migration reporting pipelines:

- `started_at`
- `first_generated_command_at`
- `time_to_first_generated_command_sec`

Primary KPI target:

- `agentcli --help` to first generated migrated command in <= 10 minutes.

## Recommended Workflow

1. Run dry-run plan first.
2. Review `summary` and per-script strategy in `plan.json`.
3. Re-run with `--apply`.
4. Resolve `manual` scripts iteratively using generated migration skill guidance.
