package dogfood

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	LedgerSchemaVersionV1 = "dogfood-ledger.v1"
	LedgerStatusOpen      = "open"
	ledgerLockTimeout     = 5 * time.Second
	ledgerLockPoll        = 10 * time.Millisecond
)

type LedgerRecord struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	Fingerprint   string    `json:"fingerprint"`
	IssueURL      string    `json:"issue_url,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type Ledger struct {
	Path string
}

func NewLedger(path string) Ledger {
	return Ledger{Path: path}
}

func (l Ledger) Append(rec LedgerRecord) error {
	if strings.TrimSpace(l.Path) == "" {
		return errors.New("ledger path is empty")
	}

	return l.withExclusiveLock(func() error {
		records, err := l.readAll()
		if err != nil {
			return err
		}

		records = append(records, normalizeLedgerRecord(rec))
		return l.writeAll(records)
	})
}

func (l Ledger) FindOpenByFingerprint(fp string) (LedgerRecord, bool, error) {
	if strings.TrimSpace(l.Path) == "" {
		return LedgerRecord{}, false, errors.New("ledger path is empty")
	}

	records, err := l.readAll()
	if err != nil {
		return LedgerRecord{}, false, err
	}

	needle := strings.TrimSpace(fp)
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		if strings.TrimSpace(rec.Fingerprint) == needle {
			if strings.EqualFold(strings.TrimSpace(rec.Status), LedgerStatusOpen) {
				return rec, true, nil
			}
			return LedgerRecord{}, false, nil
		}
	}

	return LedgerRecord{}, false, nil
}

func (l Ledger) readAll() ([]LedgerRecord, error) {
	data, err := os.ReadFile(l.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ledger %q: %w", l.Path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var records []LedgerRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode ledger %q: %w", l.Path, err)
	}

	return records, nil
}

func (l Ledger) writeAll(records []LedgerRecord) error {
	dir := filepath.Dir(l.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create ledger dir %q: %w", dir, err)
	}

	raw, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("encode ledger records: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(l.Path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp ledger file: %w", err)
	}
	tmpName := tmpFile.Name()

	cleanup := func(closeErr error) error {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return closeErr
	}

	if _, err := tmpFile.Write(raw); err != nil {
		return cleanup(fmt.Errorf("write temp ledger file: %w", err))
	}

	if err := tmpFile.Close(); err != nil {
		return cleanup(fmt.Errorf("close temp ledger file: %w", err))
	}

	if err := os.Rename(tmpName, l.Path); err != nil {
		return cleanup(fmt.Errorf("rename temp ledger file: %w", err))
	}

	return nil
}

func normalizeLedgerRecord(rec LedgerRecord) LedgerRecord {
	rec.SchemaVersion = strings.TrimSpace(rec.SchemaVersion)
	if rec.SchemaVersion == "" {
		rec.SchemaVersion = LedgerSchemaVersionV1
	}
	rec.EventID = strings.TrimSpace(rec.EventID)
	rec.Fingerprint = strings.TrimSpace(rec.Fingerprint)
	rec.IssueURL = strings.TrimSpace(rec.IssueURL)
	rec.Status = strings.TrimSpace(rec.Status)
	if !rec.CreatedAt.IsZero() {
		rec.CreatedAt = rec.CreatedAt.UTC()
	}
	return rec
}

func (l Ledger) withExclusiveLock(fn func() error) error {
	unlock, err := l.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func (l Ledger) acquireLock() (func(), error) {
	dir := filepath.Dir(l.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create ledger dir %q: %w", dir, err)
	}

	lockPath := l.Path + ".lock"
	deadline := time.Now().Add(ledgerLockTimeout)
	for {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = lock.Close()
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create ledger lock %q: %w", lockPath, err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("acquire ledger lock %q: timeout after %s", lockPath, ledgerLockTimeout)
		}
		time.Sleep(ledgerLockPoll)
	}
}
