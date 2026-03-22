package caseresource

import (
	"sort"
	"testing"

	"github.com/gh-xj/agentops/strategy"
)

func defaultTransitionsConfig() strategy.TransitionsConfig {
	return strategy.TransitionsConfig{
		Categories: map[string][]string{
			"active":    {"open", "in_progress", "blocked"},
			"completed": {"resolved", "closed_no_action"},
		},
		Initial: "open",
		Transitions: map[string]strategy.TransitionDef{
			"start":           {From: "open", To: "in_progress"},
			"block":           {From: []any{"open", "in_progress"}, To: "blocked"},
			"unblock":         {From: "blocked", To: "in_progress"},
			"resolve":         {From: []any{"in_progress", "blocked"}, To: "resolved"},
			"close_no_action": {From: []any{"open", "blocked"}, To: "closed_no_action"},
		},
	}
}

func TestStateMachineInitial(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())
	if got := sm.Initial(); got != "open" {
		t.Errorf("Initial() = %q, want %q", got, "open")
	}
}

func TestStateMachineApplyValid(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())

	tests := []struct {
		current string
		action  string
		want    string
	}{
		{"open", "start", "in_progress"},
		{"open", "block", "blocked"},
		{"in_progress", "block", "blocked"},
		{"blocked", "unblock", "in_progress"},
		{"in_progress", "resolve", "resolved"},
		{"blocked", "resolve", "resolved"},
		{"open", "close_no_action", "closed_no_action"},
		{"blocked", "close_no_action", "closed_no_action"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.action, func(t *testing.T) {
			got, err := sm.Apply(tt.current, tt.action)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Apply(%q, %q) = %q, want %q", tt.current, tt.action, got, tt.want)
			}
		})
	}
}

func TestStateMachineApplyInvalidAction(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())

	_, err := sm.Apply("open", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestStateMachineApplyInvalidFromState(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())

	// "start" only allows from "open", not "blocked"
	_, err := sm.Apply("blocked", "start")
	if err == nil {
		t.Fatal("expected error for invalid from state")
	}
}

func TestStateMachineAllStatuses(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())
	statuses := sm.AllStatuses()
	sort.Strings(statuses)

	want := []string{"blocked", "closed_no_action", "in_progress", "open", "resolved"}
	if len(statuses) != len(want) {
		t.Fatalf("AllStatuses() returned %d statuses, want %d: %v", len(statuses), len(want), statuses)
	}
	for i, s := range statuses {
		if s != want[i] {
			t.Errorf("AllStatuses()[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestStateMachineCategoryForStatus(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())

	tests := []struct {
		status string
		want   string
	}{
		{"open", "active"},
		{"in_progress", "active"},
		{"blocked", "active"},
		{"resolved", "completed"},
		{"closed_no_action", "completed"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := sm.CategoryForStatus(tt.status)
			if got != tt.want {
				t.Errorf("CategoryForStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStateMachineCategoryForStatusUnknown(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())
	got := sm.CategoryForStatus("unknown_status")
	if got != "" {
		t.Errorf("CategoryForStatus(unknown) = %q, want empty", got)
	}
}

func TestStateMachineExpandStatusFilter(t *testing.T) {
	sm := NewStateMachine(defaultTransitionsConfig())

	// Expand a category
	active, err := sm.ExpandStatusFilter("active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(active) != 3 {
		t.Errorf("active group has %d statuses, want 3", len(active))
	}

	// Expand a single status
	single, err := sm.ExpandStatusFilter("open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(single) != 1 || !single["open"] {
		t.Errorf("expected {open: true}, got %v", single)
	}

	// Unknown filter
	_, err = sm.ExpandStatusFilter("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown filter")
	}
}
