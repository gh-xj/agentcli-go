package agentcli

import (
	"runtime"
	"strings"
	"testing"
)

func TestRunCommandSuccess(t *testing.T) {
	out, err := RunCommand("sh", "-c", "printf ok")
	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}
	if out != "ok" {
		t.Fatalf("stdout mismatch: %q", out)
	}
}

func TestRunCommandFailureIncludesStderr(t *testing.T) {
	_, err := RunCommand("sh", "-c", "echo bad 1>&2; exit 7")
	if err == nil {
		t.Fatal("expected command failure")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Fatalf("expected stderr in error, got: %v", err)
	}
}

func TestWhich(t *testing.T) {
	if !Which("sh") {
		t.Fatal("expected sh to be found in PATH")
	}
	if Which("definitely-not-a-real-command-12345") {
		t.Fatal("unexpected command resolution")
	}
}

func TestRunOsascript(t *testing.T) {
	if runtime.GOOS != "darwin" || !Which("osascript") {
		t.Skip("osascript test requires macOS")
	}
	got := RunOsascript(`return "ok"`)
	if got != "ok" {
		t.Fatalf("RunOsascript mismatch: %q", got)
	}
}
