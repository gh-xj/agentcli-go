package dogfood

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventJSONIncludesRequiredKeys(t *testing.T) {
	e := Event{
		SchemaVersion: "dogfood-event.v1",
		EventID:       "evt-1",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		Timestamp:     time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"schema_version", "event_id", "event_type", "signal_source", "timestamp"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing key %s in payload %s", key, string(b))
		}
	}
}
