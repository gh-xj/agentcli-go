package harness

import (
	"errors"
	"os"
	"path/filepath"
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
