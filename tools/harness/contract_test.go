package harness

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSummarySchemaIncludesRequiredFields(t *testing.T) {
	s := CommandSummary{
		SchemaVersion: "harness.v1",
		Command:       "loop quality",
		Status:        "ok",
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	for _, key := range []string{"schema_version", "command", "status"} {
		if !strings.Contains(out, key) {
			t.Fatalf("missing key %s in %s", key, out)
		}
	}
}
