package harness

import "testing"

func TestCapabilitiesIncludesRegression(t *testing.T) {
	caps := DefaultCapabilities()
	found := false
	for _, c := range caps.Commands {
		if c.Name == "regression" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected regression command in capabilities")
	}
}
