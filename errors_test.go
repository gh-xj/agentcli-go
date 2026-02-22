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
