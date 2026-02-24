# Script Migration Hybrid Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a discoverable `agentcli migrate` workflow that converts existing `scripts/*.sh` repos into agentcli-go structure with dual modes (`safe` and `in-place`) and low cognitive load.

**Architecture:** Implement a migration pipeline (`scan -> plan -> generate -> report`) under `internal/migrate`, then wire a new CLI command in `cmd/agentcli`. Keep default behavior conservative (`safe` + wrapper-first), emit deterministic artifacts, and expose a direct AI-agent prompt in `--help`.

**Tech Stack:** Go 1.25.x, existing `agentcli` CLI structure, table-driven tests, schema contracts via `internal/tools/schemacheck`, Taskfile quality gates.

---

### Task 1: Add CLI surface for `migrate` and help discoverability

**Files:**
- Modify: `cmd/agentcli/main.go`
- Modify: `cmd/agentcli/main_test.go`

**Step 1: Write the failing test**

```go
func TestRunMigrateUsageShownInHelp(t *testing.T) {
    exitCode := run([]string{"--help"})
    if exitCode != agentcli.ExitSuccess {
        t.Fatalf("unexpected exit: %d", exitCode)
    }
    // assert help includes migrate command and one-line AI prompt
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/agentcli -run TestRunMigrateUsageShownInHelp -v`
Expected: FAIL because help output does not include migration text.

**Step 3: Write minimal implementation**

- Add `case "migrate": return runMigrate(args[1:])` in root switch.
- Add migrate usage lines in `printUsage()`.
- Add one short AI prompt starter line in help text.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/agentcli -run TestRunMigrateUsageShownInHelp -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/agentcli/main.go cmd/agentcli/main_test.go
git commit -m "feat(cli): add migrate command entry and help discoverability"
```

### Task 2: Create migration domain model and scanner

**Files:**
- Create: `internal/migrate/types.go`
- Create: `internal/migrate/scan.go`
- Create: `internal/migrate/scan_test.go`

**Step 1: Write the failing test**

```go
func TestScanScriptsDetectsBashAndSh(t *testing.T) {
    // temp repo with scripts/a.sh + scripts/b
    // assert scanner returns both with shell type, path, and basic metadata
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/migrate -run TestScanScriptsDetectsBashAndSh -v`
Expected: FAIL (package/functions missing).

**Step 3: Write minimal implementation**

- Define `ScriptInfo`, `ScanResult`, and shell enum (`bash`, `sh`, `unknown`).
- Implement scanner for shebang and extension detection.
- Capture lightweight metadata (args/env references/dependency calls).

**Step 4: Run test to verify it passes**

Run: `go test ./internal/migrate -run TestScanScriptsDetectsBashAndSh -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/migrate/types.go internal/migrate/scan.go internal/migrate/scan_test.go
git commit -m "feat(migrate): add script scanning model and detector"
```

### Task 3: Implement planner with strategy classification

**Files:**
- Create: `internal/migrate/plan.go`
- Create: `internal/migrate/plan_test.go`
- Modify: `internal/migrate/types.go`

**Step 1: Write the failing test**

```go
func TestPlanClassifiesScriptStrategies(t *testing.T) {
    // input: simple script, medium script, complex/eval script
    // expect: auto, wrapper, manual
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/migrate -run TestPlanClassifiesScriptStrategies -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

- Add `MigrationStrategy` enum (`auto`, `wrapper`, `manual`).
- Implement deterministic planner rules.
- Emit per-script reasons/risk flags.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/migrate -run TestPlanClassifiesScriptStrategies -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/migrate/types.go internal/migrate/plan.go internal/migrate/plan_test.go
git commit -m "feat(migrate): add deterministic migration planner"
```

### Task 4: Implement generator for safe/in-place outputs

**Files:**
- Create: `internal/migrate/generate.go`
- Create: `internal/migrate/generate_test.go`

**Step 1: Write the failing test**

```go
func TestGenerateSafeModeWritesParallelWorkspace(t *testing.T) {
    // assert output root differs from source root
    // assert generated command wrappers and docs/migration files exist
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/migrate -run TestGenerateSafeModeWritesParallelWorkspace -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

- Add `Generate(opts, plan)` with `ModeSafe` and `ModeInPlace`.
- Generate wrapper-first commands for `auto` and `wrapper`.
- Emit:
  - `docs/migration/report.md`
  - `docs/migration/plan.json`
  - `docs/migration/compatibility.md`
  - project-level migration skill file (e.g. `skill.migration.md`).

**Step 4: Run test to verify it passes**

Run: `go test ./internal/migrate -run TestGenerateSafeModeWritesParallelWorkspace -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/migrate/generate.go internal/migrate/generate_test.go
git commit -m "feat(migrate): add safe/in-place generation pipeline"
```

### Task 5: Add `runMigrate` command orchestration

**Files:**
- Create: `cmd/agentcli/migrate.go`
- Modify: `cmd/agentcli/main.go`
- Modify: `cmd/agentcli/main_test.go`

**Step 1: Write the failing test**

```go
func TestRunMigrateDryRunPrintsPlanWithoutWriting(t *testing.T) {
    // run migrate --dry-run
    // assert status success and no output files written
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/agentcli -run TestRunMigrateDryRunPrintsPlanWithoutWriting -v`
Expected: FAIL.

**Step 3: Write minimal implementation**

- Parse flags:
  - `--source`
  - `--mode safe|in-place`
  - `--dry-run`
  - `--apply`
  - `--out` (optional safe target override)
- Call scan/plan/generate pipeline.
- In dry-run, only print summarized plan.

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/agentcli -run TestRunMigrateDryRunPrintsPlanWithoutWriting -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add cmd/agentcli/migrate.go cmd/agentcli/main.go cmd/agentcli/main_test.go
git commit -m "feat(cli): implement migrate command flow with dual modes"
```

### Task 6: Add migration JSON contracts and schema gates

**Files:**
- Create: `schemas/migrate-plan.schema.json`
- Create: `schemas/migrate-report.schema.json`
- Create: `testdata/contracts/migrate-plan.ok.json`
- Create: `testdata/contracts/migrate-plan.bad-missing-scripts.json`
- Create: `testdata/contracts/migrate-report.ok.json`
- Create: `testdata/contracts/migrate-report.bad-missing-summary.json`
- Modify: `Taskfile.yml`

**Step 1: Write the failing test/contract gate**

- Add schema check commands in Taskfile first, expecting failure due to missing schema/files.

**Step 2: Run gate to verify failure**

Run: `task schema:check`
Expected: FAIL for missing migration schema/fixtures.

**Step 3: Write minimal implementation**

- Add schemas and positive/negative fixtures.
- Wire checks into `schema:check` and `schema:negative` tasks.

**Step 4: Run gate to verify pass**

Run: `task schema:check && task schema:negative`
Expected: PASS.

**Step 5: Commit**

```bash
git add schemas testdata/contracts Taskfile.yml
git commit -m "test(schema): add migration plan/report contract gates"
```

### Task 7: Add migration docs and KPI instrumentation

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `skills/agentcli-go/SKILL.md`
- Create: `docs/migration/README.md`

**Step 1: Write the failing test/check**

- Extend docs drift checker expectations if needed (or add CLI help test assertion for migration onboarding text).

**Step 2: Run check to verify failure**

Run: `go test ./cmd/agentcli -run TestRunMigrateUsageShownInHelp -v`
Expected: FAIL until docs/help are aligned.

**Step 3: Write minimal implementation**

- Add “migrate existing scripts” quickstart in both READMEs.
- Add KPI definition and example report fields.
- Update skill doc with migration usage guidance.

**Step 4: Run check to verify pass**

Run: `go test ./cmd/agentcli -run TestRunMigrateUsageShownInHelp -v && task docs:check`
Expected: PASS.

**Step 5: Commit**

```bash
git add README.md README.zh-CN.md skills/agentcli-go/SKILL.md docs/migration/README.md
git commit -m "docs(migrate): add migration onboarding and KPI guidance"
```

### Task 8: End-to-end verification and release readiness

**Files:**
- Modify: `CHANGELOG.md`
- Create: `docs/releases/v0.2.5.md` (or current next tag)

**Step 1: Run full verification**

Run: `task verify`
Expected: PASS.

**Step 2: Run release gate**

Run: `task release:gate`
Expected: PASS.

**Step 3: Record release notes draft**

- Add migration feature highlights, known limits, and safe-mode recommendation.

**Step 4: Commit final verification/docs**

```bash
git add CHANGELOG.md docs/releases
git commit -m "chore(release): prepare migration feature release notes"
```

