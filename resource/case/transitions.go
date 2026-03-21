package caseresource

import (
	"fmt"

	"github.com/gh-xj/agentcli-go/strategy"
)

// StateMachine enforces valid state transitions loaded from strategy config.
type StateMachine struct {
	config strategy.TransitionsConfig
}

// NewStateMachine creates a StateMachine from a TransitionsConfig.
func NewStateMachine(config strategy.TransitionsConfig) *StateMachine {
	return &StateMachine{config: config}
}

// Initial returns the initial status for new cases.
func (sm *StateMachine) Initial() string {
	return sm.config.Initial
}

// Apply applies an action to the current status and returns the new status.
func (sm *StateMachine) Apply(currentStatus, action string) (string, error) {
	def, ok := sm.config.Transitions[action]
	if !ok {
		return "", fmt.Errorf("unknown action %q", action)
	}

	fromStates := def.FromStates()
	for _, s := range fromStates {
		if s == currentStatus {
			return def.To, nil
		}
	}

	return "", fmt.Errorf("action %q not allowed from status %q (allowed from: %v)", action, currentStatus, fromStates)
}

// AllStatuses returns all known statuses from the categories config.
func (sm *StateMachine) AllStatuses() []string {
	var statuses []string
	for _, ss := range sm.config.Categories {
		statuses = append(statuses, ss...)
	}
	return statuses
}

// CategoryForStatus returns the category name for a given status.
func (sm *StateMachine) CategoryForStatus(status string) string {
	for cat, statuses := range sm.config.Categories {
		for _, s := range statuses {
			if s == status {
				return cat
			}
		}
	}
	return ""
}

// ExpandStatusFilter expands a status filter (status name or category) into a set of statuses.
func (sm *StateMachine) ExpandStatusFilter(filter string) (map[string]bool, error) {
	// Check if it's a category name.
	if statuses, ok := sm.config.Categories[filter]; ok {
		m := make(map[string]bool, len(statuses))
		for _, s := range statuses {
			m[s] = true
		}
		return m, nil
	}

	// Check if it's a known status.
	allStatuses := sm.AllStatuses()
	for _, s := range allStatuses {
		if s == filter {
			return map[string]bool{filter: true}, nil
		}
	}

	return nil, fmt.Errorf("unknown status filter %q", filter)
}
