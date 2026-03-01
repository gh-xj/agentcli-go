package dogfood

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestLedgerAppendAndFindOpenByFingerprint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dogfood-ledger.json")
	l := NewLedger(path)

	rec := LedgerRecord{
		SchemaVersion: "dogfood-ledger.v1",
		EventID:       "evt-1",
		Fingerprint:   "fp-1",
		IssueURL:      "https://github.com/o/r/issues/1",
		Status:        "open",
		CreatedAt:     time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	if err := l.Append(rec); err != nil {
		t.Fatal(err)
	}

	got, ok, err := l.FindOpenByFingerprint("fp-1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected open record, got ok=%v rec=%+v", ok, got)
	}
	if got.EventID != rec.EventID {
		t.Fatalf("expected event_id %q, got %q", rec.EventID, got.EventID)
	}
	if got.IssueURL == "" {
		t.Fatalf("expected issue_url to be set, got empty")
	}
}

func TestLedgerFindOpenByFingerprintIgnoresStaleOpenWhenLatestIsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dogfood-ledger.json")
	l := NewLedger(path)

	if err := l.Append(LedgerRecord{
		EventID:     "evt-open",
		Fingerprint: "fp-1",
		Status:      "open",
		CreatedAt:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	if err := l.Append(LedgerRecord{
		EventID:     "evt-closed",
		Fingerprint: "fp-1",
		Status:      "closed",
		CreatedAt:   time.Date(2026, 3, 1, 1, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := l.FindOpenByFingerprint("fp-1")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("expected no open record after latest close, got %+v", got)
	}
}

func TestLedgerAppendConcurrentDoesNotLoseUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dogfood-ledger.json")
	l := NewLedger(path)

	const writers = 100
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup

	for i := range writers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- l.Append(LedgerRecord{
				EventID:     fmt.Sprintf("evt-%03d", i),
				Fingerprint: "fp-concurrent",
				Status:      "open",
				CreatedAt:   time.Date(2026, 3, 1, 0, 0, i, 0, time.UTC),
			})
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("append error: %v", err)
		}
	}

	records, err := l.readAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != writers {
		t.Fatalf("expected %d records, got %d", writers, len(records))
	}
}
