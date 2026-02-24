package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Context struct {
	DryRun  bool
	Explain bool
}

type ExecuteFunc func(ctx Context) (CommandOutcome, error)

type CommandInput struct {
	Name        string
	SummaryPath string
	DryRun      bool
	Explain     bool
	Execute     ExecuteFunc
}

func Run(input CommandInput) (CommandSummary, error) {
	startedAt := time.Now().UTC()
	summary := CommandSummary{
		SchemaVersion: SummarySchemaVersion,
		Command:       strings.TrimSpace(input.Name),
		Status:        string(StatusOK),
		StartedAt:     startedAt,
	}

	if summary.Command == "" {
		err := NewFailure(CodeUsage, "command name is required", "set command name in harness input", false)
		appendFailure(&summary, FailureFromError(err))
		finalizeSummary(&summary, startedAt)
		_ = writeSummary(summary, input.SummaryPath)
		return summary, err
	}
	if input.Execute == nil {
		err := NewFailure(CodeUsage, "command executor is required", "provide Execute callback", false)
		appendFailure(&summary, FailureFromError(err))
		finalizeSummary(&summary, startedAt)
		_ = writeSummary(summary, input.SummaryPath)
		return summary, err
	}

	outcome, execErr := input.Execute(Context{
		DryRun:  input.DryRun,
		Explain: input.Explain,
	})
	summary.Checks = append(summary.Checks, outcome.Checks...)
	summary.Failures = append(summary.Failures, outcome.Failures...)
	summary.Artifacts = append(summary.Artifacts, outcome.Artifacts...)
	summary.Data = outcome.Data

	if execErr != nil {
		appendFailure(&summary, FailureFromError(execErr))
	}
	if len(summary.Failures) > 0 {
		summary.Status = string(StatusFail)
	}
	finalizeSummary(&summary, startedAt)

	if err := writeSummary(summary, input.SummaryPath); err != nil {
		writeErr := WrapFailure(CodeFileIO, "write summary", fmt.Sprintf("check summary path %q", input.SummaryPath), false, err)
		appendFailure(&summary, FailureFromError(writeErr))
		summary.Status = string(StatusFail)
		finalizeSummary(&summary, startedAt)
		return summary, writeErr
	}

	if execErr != nil {
		return summary, normalizeExecutionError(execErr)
	}
	if len(summary.Failures) > 0 {
		f := summary.Failures[0]
		return summary, &FailureError{Failure: f}
	}

	return summary, nil
}

func finalizeSummary(summary *CommandSummary, startedAt time.Time) {
	finished := time.Now().UTC()
	summary.FinishedAt = finished
	summary.DurationMs = finished.Sub(startedAt).Milliseconds()
}

func appendFailure(summary *CommandSummary, failure Failure) {
	if strings.TrimSpace(failure.Code) == "" || strings.TrimSpace(failure.Message) == "" {
		return
	}
	summary.Failures = append(summary.Failures, failure)
}

func writeSummary(summary CommandSummary, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func normalizeExecutionError(err error) error {
	if err == nil {
		return nil
	}
	if IsCode(err, CodeUsage) ||
		IsCode(err, CodeMissingDependency) ||
		IsCode(err, CodeContractValidation) ||
		IsCode(err, CodeExecution) ||
		IsCode(err, CodeFileIO) ||
		IsCode(err, CodeInternal) {
		return err
	}
	return WrapFailure(CodeExecution, "command execution failed", "", false, err)
}
