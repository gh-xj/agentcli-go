package main

import "testing"

func TestRunRequiresEventFile(t *testing.T) {
	code := run([]string{})
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
}
