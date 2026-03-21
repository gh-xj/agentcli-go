package resource

import (
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
)

// mockResource implements the Resource interface for testing.
type mockResource struct {
	kind string
}

func (m *mockResource) Schema() ResourceSchema {
	return ResourceSchema{Kind: m.kind, Description: "mock " + m.kind}
}

func (m *mockResource) Create(_ *agentcli.AppContext, _ string, _ map[string]string) (*Record, error) {
	return &Record{Kind: m.kind, ID: "new"}, nil
}

func (m *mockResource) List(_ *agentcli.AppContext, _ Filter) ([]Record, error) {
	return nil, nil
}

func (m *mockResource) Get(_ *agentcli.AppContext, id string) (*Record, error) {
	return &Record{Kind: m.kind, ID: id}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mock := &mockResource{kind: "case"}
	reg.Register(mock)

	got, ok := reg.Get("case")
	if !ok {
		t.Fatal("expected to find registered resource")
	}
	if got.Schema().Kind != "case" {
		t.Fatalf("expected kind 'case', got %q", got.Schema().Kind)
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Fatal("expected false for non-existent resource")
	}
}

func TestRegistryAllSorted(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockResource{kind: "case"})
	reg.Register(&mockResource{kind: "agent"})
	reg.Register(&mockResource{kind: "slot"})

	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(all))
	}

	want := []string{"agent", "case", "slot"}
	for i, res := range all {
		if res.Schema().Kind != want[i] {
			t.Fatalf("index %d: expected kind %q, got %q", i, want[i], res.Schema().Kind)
		}
	}
}
