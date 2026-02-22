package agentcli

import (
	"errors"
	"reflect"
	"testing"
)

type traceHook struct {
	trace *[]string
	err   error
	at    string
}

func (h traceHook) Preflight(*AppContext) error {
	*h.trace = append(*h.trace, "pre")
	if h.at == "pre" {
		return h.err
	}
	return nil
}

func (h traceHook) Postflight(*AppContext) error {
	*h.trace = append(*h.trace, "post")
	if h.at == "post" {
		return h.err
	}
	return nil
}

func TestRunLifecycleOrder(t *testing.T) {
	trace := make([]string, 0, 3)
	h := traceHook{trace: &trace}
	err := RunLifecycle(NewAppContext(nil), h, func(*AppContext) error {
		trace = append(trace, "run")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pre", "run", "post"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("trace mismatch: got %v want %v", trace, want)
	}
}

func TestRunLifecycleStopsOnPreflightError(t *testing.T) {
	trace := make([]string, 0, 3)
	wantErr := errors.New("stop")
	h := traceHook{trace: &trace, err: wantErr, at: "pre"}
	err := RunLifecycle(NewAppContext(nil), h, func(*AppContext) error {
		trace = append(trace, "run")
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected preflight error, got %v", err)
	}
	want := []string{"pre"}
	if !reflect.DeepEqual(trace, want) {
		t.Fatalf("trace mismatch: got %v want %v", trace, want)
	}
}
