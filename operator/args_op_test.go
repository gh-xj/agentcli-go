package operator

import (
	"testing"
)

func TestArgsOperator_Parse(t *testing.T) {
	op := NewArgsOperator()
	got := op.Parse([]string{"--name", "foo", "--verbose"})

	if got["name"] != "foo" {
		t.Errorf("expected name=foo, got %q", got["name"])
	}
	if got["verbose"] != "true" {
		t.Errorf("expected verbose=true, got %q", got["verbose"])
	}
}

func TestArgsOperator_Require_Success(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{"name": "foo"}

	val, err := op.Require(args, "name", "project name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "foo" {
		t.Errorf("expected foo, got %q", val)
	}
}

func TestArgsOperator_Require_Missing(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{}

	_, err := op.Require(args, "name", "project name")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestArgsOperator_Get_Existing(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{"name": "foo"}

	if got := op.Get(args, "name", "default"); got != "foo" {
		t.Errorf("expected foo, got %q", got)
	}
}

func TestArgsOperator_Get_Missing(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{}

	if got := op.Get(args, "name", "default"); got != "default" {
		t.Errorf("expected default, got %q", got)
	}
}

func TestArgsOperator_HasFlag_True(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{"verbose": "true"}

	if !op.HasFlag(args, "verbose") {
		t.Error("expected HasFlag to return true")
	}
}

func TestArgsOperator_HasFlag_False(t *testing.T) {
	op := NewArgsOperator()
	args := map[string]string{"name": "foo"}

	if op.HasFlag(args, "verbose") {
		t.Error("expected HasFlag to return false for missing key")
	}
	if op.HasFlag(args, "name") {
		t.Error("expected HasFlag to return false for non-true value")
	}
}
