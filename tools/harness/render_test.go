package harness

import (
	"strings"
	"testing"
)

func TestRenderNDJSON(t *testing.T) {
	got, err := RenderSummary(CommandSummary{
		SchemaVersion: "harness.v1",
		Command:       "loop quality",
		Status:        "ok",
	}, "ndjson", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected newline-terminated ndjson, got %q", got)
	}
}
