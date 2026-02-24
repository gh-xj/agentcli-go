# Script Migration Hybrid Design

## Problem

Current onboarding is strong for greenfield CLI scaffolding, but users with existing `scripts/*.sh` repos still need manual translation. That creates migration friction and lowers market conversion.

## Goals

- Ship a migration flow that is discoverable from `agentcli --help`.
- Support existing `bash` + POSIX `sh` scripts in v1.
- Provide dual migration modes with safe defaults:
  - `safe` (default): parallel migration workspace
  - `in-place`: direct repo conversion
- Optimize for onboarding conversion KPI:
  - from `agentcli --help` to first generated migrated command in <= 10 minutes.

## Non-Goals (v1)

- Full shell transpilation into native Go logic.
- Broad shell support beyond `bash`/`sh` (e.g., `fish`, PowerShell).
- Perfect semantic equivalence for every advanced shell construct.

## Chosen Approach

Use a hybrid flow: first-class CLI migration command plus generated migration skill/docs.

Rationale:

- CLI-first lowers cognitive load and improves discovery.
- Generated skill/docs preserve flexibility for complex repos and AI-agent follow-up.
- Safer rollout than aggressive transpilation.

## User Experience

Primary command:

```bash
agentcli migrate --source ./scripts --mode safe --dry-run
agentcli migrate --source ./scripts --mode safe --apply
```

Key UX behaviors:

- `--mode safe` is default.
- `--dry-run` prints plan + risk classification, no file writes.
- `--apply` executes generation.
- `--mode in-place` writes directly into current project structure.

Help-driven AI-agent onboarding:

- `agentcli --help` includes one migration call-to-action and one agent prompt line.
- `agentcli migrate --help` includes a short, deterministic runbook.

## Architecture

Pipeline:

1. `scan`: parse candidate scripts and extract metadata.
2. `plan`: classify scripts by migration strategy.
3. `generate`: scaffold commands and migration artifacts.
4. `report`: emit machine-readable + human-readable outputs.

Suggested package layout:

- `internal/migrate/types.go`
- `internal/migrate/scan.go`
- `internal/migrate/plan.go`
- `internal/migrate/generate.go`
- `internal/migrate/report.go`

CLI entrypoint:

- `cmd/agentcli/migrate.go` and `runMigrate(args []string)` in command routing.

## Script Classification Model

Each script gets a strategy:

- `auto`: straightforward mapping to generated command wrapper with typed flags.
- `wrapper`: shell script kept, generated command invokes it with guarded env/args.
- `manual`: unsupported patterns; emit explicit TODO and migration notes.

Risk factors (examples):

- `eval`, dynamic sourcing, process substitution, heavy trap logic, complex heredoc chains.

## Generated Outputs

Always generate:

- `docs/migration/report.md` (human summary, migration status, next steps)
- `docs/migration/plan.json` (machine-readable plan)
- `docs/migration/compatibility.md` (detected shell features and confidence)

When `--apply`:

- scaffolded command files in target location (`safe` or `in-place`)
- migration helper skill file (project-level), e.g. `skill.migration.md`

## Modes

### Safe Mode (default)

- Generates a parallel target tree (e.g. `<repo>/agentcli-migrated/` or configured output).
- Leaves existing scripts untouched.
- Designed for review and incremental cutover.

### In-place Mode

- Writes command outputs into current repo structure.
- Optional legacy script tagging/deprecation notes in report.
- Requires explicit confirmation flag (`--apply`).

## Error Handling

- Fail-fast on invalid paths, unreadable scripts, and write permission failures.
- Partial migration is allowed; report must always include per-script status.
- Return deterministic exit codes for:
  - usage/config errors
  - scan/parse errors
  - generation failures
  - partial/manual-required outcomes

## Quality & Verification

Test strategy:

- unit tests for scan/plan classification and flag extraction
- golden tests for generated command wrappers and migration docs
- CLI tests for mode behavior, dry-run/apply behavior, and help output
- schema validation for `plan.json` and optional report JSON

CI updates:

- add migration schema checks in existing `schema:check`/`schema:negative`
- add migration smoke fixture for a representative `scripts/` sample

## KPI Instrumentation

Record onboarding conversion KPI in migration report:

- `started_at`
- `first_generated_command_at`
- `time_to_first_generated_command_sec`

Initial success target:

- >= 80% of first-time users complete first migration command generation in <= 10 minutes.

## Rollout Plan

1. Internal alpha on representative script repos.
2. Public beta with `safe` mode promoted first.
3. Add case-study docs showing before/after migration path.
4. Keep `in-place` available but explicitly marked advanced.

## Tradeoffs Accepted

- v1 prioritizes onboarding conversion over deep shell semantic coverage.
- wrapper-first migration may generate conservative outputs, but preserves reliability.
- dual mode adds CLI surface area, mitigated by strong help text and safe default.
