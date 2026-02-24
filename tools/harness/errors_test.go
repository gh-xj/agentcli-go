package harness

import "testing"

func TestExitCodeMapping(t *testing.T) {
	if code := ExitCodeFor(NewFailure(CodeUsage, "bad args", "check --help", false)); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if code := ExitCodeFor(NewFailure(CodeMissingDependency, "missing binary", "install foo", false)); code != 3 {
		t.Fatalf("expected 3, got %d", code)
	}
	if code := ExitCodeFor(NewFailure(CodeContractValidation, "drift", "update baseline", false)); code != 4 {
		t.Fatalf("expected 4, got %d", code)
	}
	if code := ExitCodeFor(NewFailure(CodeExecution, "failed", "", false)); code != 5 {
		t.Fatalf("expected 5, got %d", code)
	}
	if code := ExitCodeFor(NewFailure(CodeFileIO, "io", "", false)); code != 6 {
		t.Fatalf("expected 6, got %d", code)
	}
	if code := ExitCodeFor(NewFailure(CodeInternal, "panic", "", false)); code != 7 {
		t.Fatalf("expected 7, got %d", code)
	}
}
