package agentcli

import (
	"errors"
	"testing"
)

func TestResolveExitCode(t *testing.T) {
	if got := ResolveExitCode(nil); got != ExitSuccess {
		t.Fatalf("nil error expected %d, got %d", ExitSuccess, got)
	}
	if got := ResolveExitCode(errors.New("x")); got != ExitFailure {
		t.Fatalf("plain error expected %d, got %d", ExitFailure, got)
	}

	err := NewCLIError(ExitUsage, "usage", "invalid input", nil)
	if got := ResolveExitCode(err); got != ExitUsage {
		t.Fatalf("typed error expected %d, got %d", ExitUsage, got)
	}
}

func TestCLIErrorWrap(t *testing.T) {
	cause := errors.New("root")
	err := NewCLIError(ExitRuntimeExternal, "runtime", "failed command", cause)
	if !errors.Is(err, cause) {
		t.Fatal("expected wrapped cause")
	}
	if got := err.ExitCode(); got != ExitRuntimeExternal {
		t.Fatalf("exit code mismatch: %d", got)
	}
}

func TestAgentopsExitCodes(t *testing.T) {
	codes := []struct {
		name string
		code int
		want int
	}{
		{"StrategyMissing", ExitStrategyMissing, 10},
		{"TransitionDenied", ExitTransitionDenied, 11},
		{"WorkerFailed", ExitWorkerFailed, 12},
		{"ValidationFailed", ExitValidationFailed, 13},
	}
	for _, tc := range codes {
		t.Run(tc.name, func(t *testing.T) {
			if tc.code != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, tc.code)
			}
			err := NewCLIError(tc.code, tc.name, "test", nil)
			if got := ResolveExitCode(err); got != tc.want {
				t.Fatalf("ResolveExitCode: expected %d, got %d", tc.want, got)
			}
		})
	}
}
