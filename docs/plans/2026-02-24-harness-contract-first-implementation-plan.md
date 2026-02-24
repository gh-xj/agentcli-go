# Harness Contract-First Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace loop runtime behavior with a contract-first harness core (`tools/harness`) and make `agentcli loop` a thin adapter with deterministic machine-friendly outputs.

**Architecture:** Introduce a new canonical harness engine under `tools/harness` with shared contract, error taxonomy, and runner wrapper. Migrate `cmd/agentcli/loop.go` to delegate command behavior to this engine, preserving the CLI surface but removing legacy runtime branching from adapter code. Enforce behavior-only regression and summary schema stability through tests and CI gates.

**Tech Stack:** Go (`go test`, `gofmt`), existing `agentcli` CLI entrypoints, Taskfile CI pipeline, JSON schema + golden contract fixtures.

---

### Task 1: Scaffold `tools/harness` Contract Package

**Files:**
- Create: `tools/harness/contract.go`
- Create: `tools/harness/contract_test.go`

**Step 1: Write the failing test**

```go
func TestSummarySchemaIncludesRequiredFields(t *testing.T) {
    s := CommandSummary{
        SchemaVersion: "harness.v1",
        Command:       "loop quality",
        Status:        "ok",
    }
    b, err := json.Marshal(s)
    if err != nil {
        t.Fatal(err)
    }
    out := string(b)
    for _, key := range []string{"schema_version", "command", "status"} {
        if !strings.Contains(out, key) {
            t.Fatalf("missing key %s in %s", key, out)
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness -run TestSummarySchemaIncludesRequiredFields -v`
Expected: FAIL (package or symbols not found)

**Step 3: Write minimal implementation**

```go
type CommandSummary struct {
    SchemaVersion string `json:"schema_version"`
    Command       string `json:"command"`
    Status        string `json:"status"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness -run TestSummarySchemaIncludesRequiredFields -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/contract.go tools/harness/contract_test.go
git commit -m "feat(harness): add base command summary contract"
```

### Task 2: Add Typed Failure Envelope + Exit Code Mapping

**Files:**
- Create: `tools/harness/errors.go`
- Create: `tools/harness/errors_test.go`

**Step 1: Write the failing test**

```go
func TestExitCodeMapping(t *testing.T) {
    if code := ExitCodeFor(NewFailure(CodeUsage, "bad args", "check --help", false)); code != 2 {
        t.Fatalf("expected 2, got %d", code)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness -run TestExitCodeMapping -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
const (
    ExitSuccess = 0
    ExitUsage   = 2
    ExitMissingDependency = 3
    ExitContractFailure   = 4
    ExitExecutionFailure  = 5
    ExitIOFailure         = 6
    ExitInternalFailure   = 7
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness -run TestExitCodeMapping -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/errors.go tools/harness/errors_test.go
git commit -m "feat(harness): add typed failures and deterministic exit mapping"
```

### Task 3: Build Shared Runner (Always Emits Summary)

**Files:**
- Create: `tools/harness/runner.go`
- Create: `tools/harness/runner_test.go`

**Step 1: Write the failing test**

```go
func TestRunnerWritesSummaryOnFailure(t *testing.T) {
    summaryPath := filepath.Join(t.TempDir(), "summary.json")
    _, err := Run(CommandInput{
        Name:        "loop regression",
        SummaryPath: summaryPath,
        Execute: func(ctx Context) ([]CheckResult, []Failure, []string, error) {
            return nil, []Failure{{Code: CodeExecution, Message: "boom"}}, nil, errors.New("boom")
        },
    })
    if err == nil {
        t.Fatal("expected error")
    }
    if _, statErr := os.Stat(summaryPath); statErr != nil {
        t.Fatalf("expected summary file: %v", statErr)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness -run TestRunnerWritesSummaryOnFailure -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
func Run(input CommandInput) (CommandSummary, error) {
    // measure timings, call Execute, map typed failures, always write summary
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness -run TestRunnerWritesSummaryOnFailure -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/runner.go tools/harness/runner_test.go
git commit -m "feat(harness): add shared runner with summary-on-failure"
```

### Task 4: Implement Output Renderers (`text|json|ndjson`)

**Files:**
- Modify: `tools/harness/contract.go`
- Create: `tools/harness/render.go`
- Create: `tools/harness/render_test.go`

**Step 1: Write the failing test**

```go
func TestRenderNDJSON(t *testing.T) {
    got, err := RenderSummary(CommandSummary{SchemaVersion: "harness.v1", Command: "loop quality", Status: "ok"}, "ndjson", false)
    if err != nil {
        t.Fatal(err)
    }
    if !strings.HasSuffix(got, "\n") {
        t.Fatalf("expected newline-terminated ndjson, got %q", got)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness -run TestRenderNDJSON -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
func RenderSummary(s CommandSummary, format string, noColor bool) (string, error) { ... }
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness -run TestRenderNDJSON -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/contract.go tools/harness/render.go tools/harness/render_test.go
git commit -m "feat(harness): add summary renderers for text/json/ndjson"
```

### Task 5: Add Capability Discovery Command Model

**Files:**
- Create: `tools/harness/capabilities.go`
- Create: `tools/harness/capabilities_test.go`

**Step 1: Write the failing test**

```go
func TestCapabilitiesIncludesRegression(t *testing.T) {
    caps := DefaultCapabilities()
    found := false
    for _, c := range caps.Commands {
        if c.Name == "regression" {
            found = true
            break
        }
    }
    if !found {
        t.Fatal("expected regression command in capabilities")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness -run TestCapabilitiesIncludesRegression -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
type Capabilities struct { ... }
func DefaultCapabilities() Capabilities { ... }
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness -run TestCapabilitiesIncludesRegression -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/capabilities.go tools/harness/capabilities_test.go
git commit -m "feat(harness): add machine-readable capabilities model"
```

### Task 6: Build Harness Command Modules (Doctor/Quality/Lean/Regression/Lab)

**Files:**
- Create: `tools/harness/commands/doctor.go`
- Create: `tools/harness/commands/quality.go`
- Create: `tools/harness/commands/lean.go`
- Create: `tools/harness/commands/regression.go`
- Create: `tools/harness/commands/lab.go`
- Create: `tools/harness/commands/commands_test.go`

**Step 1: Write the failing test**

```go
func TestRegressionCommandReturnsContractFailureOnDrift(t *testing.T) {
    cmd := NewRegressionCommand()
    _, err := cmd.Run(Context{/* baseline with drift */})
    if err == nil {
        t.Fatal("expected drift error")
    }
    if !IsCode(err, CodeContractValidation) {
        t.Fatalf("expected contract validation code, got %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness/commands -run TestRegressionCommandReturnsContractFailureOnDrift -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
type Command interface { Name() string; Run(ctx harness.Context) (...)}
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness/commands -run TestRegressionCommandReturnsContractFailureOnDrift -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/commands/*.go
git commit -m "feat(harness): add command modules for doctor quality lean regression lab"
```

### Task 7: Rewrite `cmd/agentcli/loop.go` as Adapter-Only

**Files:**
- Modify: `cmd/agentcli/loop.go`
- Modify: `cmd/agentcli/main.go`
- Modify: `cmd/agentcli/main_test.go`

**Step 1: Write the failing test**

```go
func TestLoopHelpIncludesRegressionAndCapabilities(t *testing.T) {
    out := captureHelpOutput(t)
    if !strings.Contains(out, "regression") || !strings.Contains(out, "capabilities") {
        t.Fatalf("missing command in help: %s", out)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/agentcli -run TestLoopHelpIncludesRegressionAndCapabilities -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// Parse global flags once, resolve command, call tools/harness dispatcher.
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/agentcli -run TestLoopHelpIncludesRegressionAndCapabilities -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/agentcli/loop.go cmd/agentcli/main.go cmd/agentcli/main_test.go
git commit -m "refactor(loop): delegate runtime behavior to tools/harness adapter"
```

### Task 8: Preserve Behavior-Only Regression Baseline Contract

**Files:**
- Modify: `tools/harness/commands/regression.go`
- Modify: `testdata/regression/loop-quality.behavior-baseline.json`
- Create: `tools/harness/commands/regression_test.go`

**Step 1: Write the failing test**

```go
func TestRegressionUsesBehaviorOnlySnapshot(t *testing.T) {
    // ensure perf fields are ignored and behavior fields are compared
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./tools/harness/commands -run TestRegressionUsesBehaviorOnlySnapshot -v`
Expected: FAIL

**Step 3: Write minimal implementation**

```go
// Compare scenario step exits, findings tuple, judge envelope, committee strategy.
```

**Step 4: Run test to verify it passes**

Run: `go test ./tools/harness/commands -run TestRegressionUsesBehaviorOnlySnapshot -v`
Expected: PASS

**Step 5: Commit**

```bash
git add tools/harness/commands/regression.go tools/harness/commands/regression_test.go testdata/regression/loop-quality.behavior-baseline.json
git commit -m "feat(regression): enforce behavior-only drift contract"
```

### Task 9: Add Summary Schema Stability and Failure Coverage Tests

**Files:**
- Create: `testdata/contracts/harness-summary.ok.json`
- Create: `testdata/contracts/harness-summary.bad-missing-status.json`
- Create: `schemas/harness-summary.schema.json`
- Modify: `Taskfile.yml`

**Step 1: Write the failing schema check command**

Run: `go run ./internal/tools/schemacheck --schema schemas/harness-summary.schema.json --input testdata/contracts/harness-summary.ok.json`
Expected: FAIL (files missing)

**Step 2: Add schema + fixtures**

```json
{"schema_version":"harness.v1","command":"loop regression","status":"ok"}
```

**Step 3: Verify positive and negative schema checks**

Run:
- `go run ./internal/tools/schemacheck --schema schemas/harness-summary.schema.json --input testdata/contracts/harness-summary.ok.json`
- `sh -c '! go run ./internal/tools/schemacheck --schema schemas/harness-summary.schema.json --input testdata/contracts/harness-summary.bad-missing-status.json'`
Expected: first PASS, second FAIL (as expected)

**Step 4: Wire into Taskfile**

Run: `task ci`
Expected: PASS including new harness summary schema checks.

**Step 5: Commit**

```bash
git add schemas/harness-summary.schema.json testdata/contracts/harness-summary.*.json Taskfile.yml
git commit -m "test(harness): add summary schema contract checks"
```

### Task 10: Documentation and Operator Workflow Update

**Files:**
- Modify: `README.md`
- Modify: `agents.md`
- Modify: `skills/verification-loop/SKILL.md`
- Modify: `skills/loop-governance/SKILL.md`

**Step 1: Write doc drift test expectation**

Run: `go run ./internal/tools/doccheck --repo-root .`
Expected: FAIL if signatures changed and docs not updated.

**Step 2: Update docs with new contract flags and capabilities**

```text
agentcli loop capabilities --format json
agentcli loop regression --write-baseline
--format --summary --dry-run --explain
```

**Step 3: Run drift + CI checks**

Run:
- `go run ./internal/tools/doccheck --repo-root .`
- `task ci`
Expected: PASS

**Step 4: Commit**

```bash
git add README.md agents.md skills/verification-loop/SKILL.md skills/loop-governance/SKILL.md
git commit -m "docs(harness): document contract-first loop workflow"
```

### Task 11: Remove Dead Legacy Runtime Paths

**Files:**
- Modify/Delete: `internal/harnessloop/*` (only where superseded)
- Modify: `cmd/agentcli/loop.go`
- Modify: affected tests under `internal/harnessloop/*_test.go`

**Step 1: Write failing compile/test check**

Run: `go test ./...`
Expected: FAIL due stale references after adapter cutover.

**Step 2: Remove or isolate dead code safely**

```go
// Keep only types/utils still used by non-loop consumers; delete superseded runtime paths.
```

**Step 3: Re-run full verification**

Run:
- `go test ./...`
- `task ci`
Expected: PASS

**Step 4: Commit**

```bash
git add -A
git commit -m "refactor(harness): remove superseded legacy loop runtime paths"
```

### Task 12: Final Integration Verification

**Files:**
- Modify: none (verification-only unless failures found)

**Step 1: End-to-end command checks**

Run:
- `go run ./cmd/agentcli loop capabilities --format json`
- `go run ./cmd/agentcli loop quality --repo-root . --format json`
- `go run ./cmd/agentcli loop regression --repo-root . --format json`
- `go run ./cmd/agentcli loop regression --repo-root . --write-baseline --format json`

Expected: all commands return summary contract with expected exit codes.

**Step 2: Full repo validation**

Run:
- `go test ./...`
- `task ci`

Expected: PASS

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat(harness): complete contract-first aggressive cutover"
```

## Notes for Execution

- Apply DRY/YAGNI: do not duplicate output logic in command modules.
- Keep all error paths typed through one envelope.
- Keep summary generation centralized in `tools/harness/runner.go`.
- Ensure all command paths emit summary payload regardless of success/failure.
