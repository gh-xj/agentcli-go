package harness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerWritesSummaryOnFailure(t *testing.T) {
	summaryPath := filepath.Join(t.TempDir(), "summary.json")
	_, err := Run(CommandInput{
		Name:        "loop regression",
		SummaryPath: summaryPath,
		Execute: func(ctx Context) (CommandOutcome, error) {
			return CommandOutcome{
				Failures: []Failure{
					{
						Code:      string(CodeExecution),
						Message:   "boom",
						Retryable: false,
					},
				},
			}, errors.New("boom")
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(summaryPath); statErr != nil {
		t.Fatalf("expected summary file: %v", statErr)
	}
}

func TestRunnerWriteSummaryFailureKeepsExecutionCode(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "not-a-dir")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	summaryPath := filepath.Join(blocker, "summary.json")

	_, err := Run(CommandInput{
		Name:        "loop run",
		SummaryPath: summaryPath,
		Execute: func(ctx Context) (CommandOutcome, error) {
			return CommandOutcome{}, NewFailure(CodeExecution, "boom", "", false)
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if ExitCodeFor(err) != ExitExecutionFailure {
		t.Fatalf("expected execution exit code, got %d", ExitCodeFor(err))
	}
	if !strings.Contains(err.Error(), "summary write also failed") {
		t.Fatalf("expected combined error context, got %v", err)
	}
}
