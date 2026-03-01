package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gh-xj/agentcli-go/internal/dogfood"
)

func TestRunRequiresEventFile(t *testing.T) {
	code := run([]string{})
	if code != 2 {
		t.Fatalf("expected usage exit code 2, got %d", code)
	}
}

func TestRunQueueRetryWithIssueURLDoesNotCreateDuplicate(t *testing.T) {
	eventPath := writeEventFixture(t)
	ledger := &fakeLedger{
		findRecord: dogfood.LedgerRecord{
			Fingerprint: "fp",
			IssueURL:    "https://github.com/gh-xj/agentcli-go/issues/101",
			Status:      string(dogfood.ActionQueueRetry),
		},
		findOK: true,
	}
	publisher := &fakePublisher{results: []publishResult{{
		url:    "https://github.com/gh-xj/agentcli-go/issues/101",
		action: dogfood.PublishActionCommented,
	}}}
	markers := &memoryIdempotencyStore{values: map[string]string{}}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := runWithDeps([]string{"--event", eventPath, "--ledger", filepath.Join(t.TempDir(), "ledger.json"), "--repo", "gh-xj/agentcli-go"}, runtimeDeps{
		stdout: stdout,
		stderr: stderr,
		getwd: func() (string, error) {
			return t.TempDir(), nil
		},
		readGitRemote: func(string) string { return "" },
		loadEvent:     loadEvent,
		now: func() time.Time {
			return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		},
		newLedger:        func(string) ledgerStore { return ledger },
		publisher:        publisher,
		newIdempotencyDB: func(string) idempotencyStore { return markers },
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (stderr=%s)", code, stderr.String())
	}

	if len(publisher.calls) != 1 {
		t.Fatalf("expected one publish call, got %d", len(publisher.calls))
	}
	if publisher.calls[0].ExistingIssueURL != "https://github.com/gh-xj/agentcli-go/issues/101" {
		t.Fatalf("expected comment path to existing issue, got %q", publisher.calls[0].ExistingIssueURL)
	}
}

func TestRunUsesIdempotencyMarkerWhenLedgerAppendFails(t *testing.T) {
	eventPath := writeEventFixture(t)
	markerStore := &memoryIdempotencyStore{values: map[string]string{}}
	publisher := &fakePublisher{results: []publishResult{
		{url: "https://github.com/gh-xj/agentcli-go/issues/222", action: dogfood.PublishActionCreated},
		{url: "https://github.com/gh-xj/agentcli-go/issues/222", action: dogfood.PublishActionCommented},
	}}

	firstLedger := &fakeLedger{appendErr: errors.New("disk full")}
	secondLedger := &fakeLedger{}
	ledgers := []ledgerStore{firstLedger, secondLedger}

	newLedger := func(string) ledgerStore {
		l := ledgers[0]
		ledgers = ledgers[1:]
		return l
	}

	deps := runtimeDeps{
		stdout: new(bytes.Buffer),
		stderr: new(bytes.Buffer),
		getwd: func() (string, error) {
			return t.TempDir(), nil
		},
		readGitRemote: func(string) string { return "" },
		loadEvent:     loadEvent,
		now: func() time.Time {
			return time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		},
		newLedger:        newLedger,
		publisher:        publisher,
		newIdempotencyDB: func(string) idempotencyStore { return markerStore },
	}

	args := []string{"--event", eventPath, "--ledger", filepath.Join(t.TempDir(), "ledger.json"), "--repo", "gh-xj/agentcli-go"}
	if code := runWithDeps(args, deps); code != 1 {
		t.Fatalf("expected first run to fail on ledger append, got %d", code)
	}

	if len(markerStore.values) != 1 {
		t.Fatalf("expected exactly one idempotency marker entry, got %d", len(markerStore.values))
	}
	for _, gotURL := range markerStore.values {
		if gotURL != "https://github.com/gh-xj/agentcli-go/issues/222" {
			t.Fatalf("expected marker issue url to be persisted, got %q", gotURL)
		}
	}

	if code := runWithDeps(args, deps); code != 0 {
		t.Fatalf("expected second run to succeed via marker dedupe, got %d", code)
	}

	if len(publisher.calls) != 2 {
		t.Fatalf("expected two publish calls, got %d", len(publisher.calls))
	}
	if publisher.calls[0].ExistingIssueURL != "" {
		t.Fatalf("expected first call to create issue, got existing issue %q", publisher.calls[0].ExistingIssueURL)
	}
	if publisher.calls[1].ExistingIssueURL != "https://github.com/gh-xj/agentcli-go/issues/222" {
		t.Fatalf("expected second call to comment existing issue from marker, got %q", publisher.calls[1].ExistingIssueURL)
	}
}

func TestFileIdempotencyStoreConcurrentPutDoesNotLoseUpdates(t *testing.T) {
	store := fileIdempotencyStore{Path: filepath.Join(t.TempDir(), "marker.json")}

	const writers = 64
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup

	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- store.Put(
				fmt.Sprintf("fp-%03d", i),
				fmt.Sprintf("https://github.com/gh-xj/agentcli-go/issues/%d", 1000+i),
			)
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("put returned error: %v", err)
		}
	}

	for i := range writers {
		key := fmt.Sprintf("fp-%03d", i)
		wantURL := fmt.Sprintf("https://github.com/gh-xj/agentcli-go/issues/%d", 1000+i)
		gotURL, ok, err := store.Get(key)
		if err != nil {
			t.Fatalf("get %q returned error: %v", key, err)
		}
		if !ok {
			t.Fatalf("missing idempotency marker for %q", key)
		}
		if gotURL != wantURL {
			t.Fatalf("unexpected marker value for %q: got %q want %q", key, gotURL, wantURL)
		}
	}
}

func TestFileIdempotencyStorePutRecoversStaleLock(t *testing.T) {
	store := fileIdempotencyStore{Path: filepath.Join(t.TempDir(), "marker.json")}
	lockPath := store.Path + ".lock"

	if err := os.WriteFile(lockPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("create lock file: %v", err)
	}

	staleTime := time.Now().Add(-idempotencyStaleLock - time.Second)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("mark lock file stale: %v", err)
	}

	if err := store.Put("fp-stale", "https://github.com/gh-xj/agentcli-go/issues/999"); err != nil {
		t.Fatalf("put with stale lock should recover, got error: %v", err)
	}

	gotURL, ok, err := store.Get("fp-stale")
	if err != nil {
		t.Fatalf("get marker after stale lock recovery: %v", err)
	}
	if !ok {
		t.Fatalf("expected marker to exist after stale lock recovery")
	}
	if gotURL != "https://github.com/gh-xj/agentcli-go/issues/999" {
		t.Fatalf("unexpected marker url: got %q", gotURL)
	}
}

type fakeLedger struct {
	findRecord dogfood.LedgerRecord
	findOK     bool
	findErr    error

	appendErr error
	appended  []dogfood.LedgerRecord
}

func (f *fakeLedger) FindOpenByFingerprint(string) (dogfood.LedgerRecord, bool, error) {
	return f.findRecord, f.findOK, f.findErr
}

func (f *fakeLedger) Append(rec dogfood.LedgerRecord) error {
	f.appended = append(f.appended, rec)
	return f.appendErr
}

type publishResult struct {
	url    string
	action string
	err    error
}

type fakePublisher struct {
	results []publishResult
	calls   []dogfood.PublishInput
}

func (f *fakePublisher) Publish(in dogfood.PublishInput) (string, string, error) {
	f.calls = append(f.calls, in)
	if len(f.results) == 0 {
		return "", "", errors.New("unexpected publish call")
	}
	res := f.results[0]
	f.results = f.results[1:]
	return res.url, res.action, res.err
}

type memoryIdempotencyStore struct {
	values map[string]string
}

func (m *memoryIdempotencyStore) Get(fingerprint string) (string, bool, error) {
	if m.values == nil {
		m.values = map[string]string{}
	}
	url, ok := m.values[fingerprint]
	return url, ok, nil
}

func (m *memoryIdempotencyStore) Put(fingerprint, issueURL string) error {
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[fingerprint] = issueURL
	return nil
}

func writeEventFixture(t *testing.T) string {
	t.Helper()
	e := dogfood.Event{
		SchemaVersion: "dogfood-event.v1",
		EventID:       "evt-1",
		EventType:     dogfood.EventTypeRuntimeError,
		SignalSource:  "local",
		Timestamp:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		ErrorSummary:  "panic: boom",
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
