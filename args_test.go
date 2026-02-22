package agentcli

import "testing"

func TestParseArgsBasic(t *testing.T) {
	got := ParseArgs([]string{"--src", "/tmp/src", "--dry-run", "--dest", "/tmp/dst"})
	if got["src"] != "/tmp/src" {
		t.Fatalf("src mismatch: %q", got["src"])
	}
	if got["dry-run"] != "true" {
		t.Fatalf("dry-run flag mismatch: %q", got["dry-run"])
	}
	if got["dest"] != "/tmp/dst" {
		t.Fatalf("dest mismatch: %q", got["dest"])
	}
}

func TestGetArgAndHasFlag(t *testing.T) {
	args := map[string]string{
		"mode": "safe",
		"yes":  "true",
	}

	if got := GetArg(args, "mode", "default"); got != "safe" {
		t.Fatalf("mode mismatch: %q", got)
	}
	if got := GetArg(args, "missing", "default"); got != "default" {
		t.Fatalf("default mismatch: %q", got)
	}
	if !HasFlag(args, "yes") {
		t.Fatal("expected yes flag to be true")
	}
	if HasFlag(args, "mode") {
		t.Fatal("expected mode to not be treated as a boolean flag")
	}
}
