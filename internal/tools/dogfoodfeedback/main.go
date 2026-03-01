package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-xj/agentcli-go/internal/dogfood"
)

const defaultLedgerPath = ".docs/dogfood/ledger.json"

const (
	idempotencyLockTimeout = 5 * time.Second
	idempotencyLockPoll    = 10 * time.Millisecond
	idempotencyStaleLock   = 30 * time.Second
)

type ledgerStore interface {
	FindOpenByFingerprint(fp string) (dogfood.LedgerRecord, bool, error)
	Append(rec dogfood.LedgerRecord) error
}

type issuePublisher interface {
	Publish(in dogfood.PublishInput) (issueURL, action string, err error)
}

type idempotencyStore interface {
	Get(fingerprint string) (issueURL string, ok bool, err error)
	Put(fingerprint, issueURL string) error
}

type runtimeDeps struct {
	stdout io.Writer
	stderr io.Writer

	getwd            func() (string, error)
	readGitRemote    func(cwd string) string
	loadEvent        func(path string) (dogfood.Event, error)
	now              func() time.Time
	newLedger        func(path string) ledgerStore
	publisher        issuePublisher
	newIdempotencyDB func(ledgerPath string) idempotencyStore
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	return runWithDeps(args, defaultRuntimeDeps())
}

func runWithDeps(args []string, deps runtimeDeps) int {
	deps = deps.withDefaults()

	fs := flag.NewFlagSet("dogfoodfeedback", flag.ContinueOnError)
	fs.SetOutput(deps.stderr)

	eventPath := fs.String("event", "", "path to dogfood event json")
	ledgerPath := fs.String("ledger", defaultLedgerPath, "path to dogfood ledger json")
	overrideRepo := fs.String("repo", "", "override target repo (owner/name)")
	dryRun := fs.Bool("dry-run", false, "print decision without publishing")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*eventPath) == "" {
		fmt.Fprintln(deps.stderr, "--event is required")
		fs.Usage()
		return 2
	}

	event, err := deps.loadEvent(*eventPath)
	if err != nil {
		fmt.Fprintf(deps.stderr, "load event: %v\n", err)
		return 1
	}

	cwd, err := deps.getwd()
	if err != nil {
		fmt.Fprintf(deps.stderr, "resolve cwd: %v\n", err)
		return 1
	}

	route := dogfood.Router{}.Resolve(dogfood.RouteInput{
		OverrideRepo: strings.TrimSpace(*overrideRepo),
		CWD:          cwd,
		GitRemote:    deps.readGitRemote(cwd),
	})
	if strings.TrimSpace(event.RepoGuess) == "" {
		event.RepoGuess = route.Repo
	}

	fp := dogfood.Fingerprint(event)
	ledger := deps.newLedger(*ledgerPath)
	idempotency := deps.newIdempotencyDB(*ledgerPath)

	existingIssueURL, err := resolveExistingIssueURL(fp, ledger, idempotency)
	if err != nil {
		fmt.Fprintf(deps.stderr, "resolve existing issue: %v\n", err)
		return 1
	}

	decision := dogfood.Engine{MinPublishConfidence: dogfood.DefaultMinConfidence}.Decide(dogfood.DecisionInput{
		RepoConfidence: route.Confidence,
		HasOpenIssue:   strings.TrimSpace(existingIssueURL) != "",
		Fingerprint:    fp,
	})

	if *dryRun {
		fmt.Fprintf(deps.stdout, "decision=%s reason=%s repo=%s confidence=%.2f fingerprint=%s existing_issue=%s\n", decision.Action, decision.Reason, route.Repo, route.Confidence, fp, existingIssueURL)
		return 0
	}

	now := deps.now().UTC()
	if decision.Action == dogfood.ActionPendingReview {
		if err := ledger.Append(dogfood.LedgerRecord{
			SchemaVersion: dogfood.LedgerSchemaVersionV1,
			EventID:       strings.TrimSpace(event.EventID),
			Fingerprint:   fp,
			Status:        string(dogfood.ActionPendingReview),
			CreatedAt:     now,
		}); err != nil {
			fmt.Fprintf(deps.stderr, "append pending ledger record: %v\n", err)
			return 1
		}
		fmt.Fprintf(deps.stdout, "decision=%s reason=%s fingerprint=%s\n", decision.Action, decision.Reason, fp)
		return 0
	}

	publishInput := dogfood.PublishInput{
		Repo:             route.Repo,
		Title:            issueTitle(event),
		Body:             issueBody(event, route, fp),
		ExistingIssueURL: existingIssueURL,
	}

	issueURL, publishAction, err := deps.publisher.Publish(publishInput)
	if err != nil {
		_ = ledger.Append(dogfood.LedgerRecord{
			SchemaVersion: dogfood.LedgerSchemaVersionV1,
			EventID:       strings.TrimSpace(event.EventID),
			Fingerprint:   fp,
			IssueURL:      strings.TrimSpace(existingIssueURL),
			Status:        string(dogfood.ActionQueueRetry),
			CreatedAt:     now,
		})
		fmt.Fprintf(deps.stderr, "publish feedback: %v\n", err)
		return 1
	}

	markerErr := idempotency.Put(fp, issueURL)
	if markerErr != nil {
		fmt.Fprintf(deps.stderr, "persist idempotency marker: %v\n", markerErr)
	}

	if err := ledger.Append(dogfood.LedgerRecord{
		SchemaVersion: dogfood.LedgerSchemaVersionV1,
		EventID:       strings.TrimSpace(event.EventID),
		Fingerprint:   fp,
		IssueURL:      issueURL,
		Status:        dogfood.LedgerStatusOpen,
		CreatedAt:     now,
	}); err != nil {
		fmt.Fprintf(deps.stderr, "append open ledger record: %v\n", err)
		return 1
	}

	fmt.Fprintf(deps.stdout, "action=%s issue=%s fingerprint=%s\n", publishAction, issueURL, fp)
	return 0
}

func defaultRuntimeDeps() runtimeDeps {
	return runtimeDeps{
		stdout:        os.Stdout,
		stderr:        os.Stderr,
		getwd:         os.Getwd,
		readGitRemote: readGitRemote,
		loadEvent:     loadEvent,
		now:           time.Now,
		newLedger:     func(path string) ledgerStore { return dogfood.NewLedger(path) },
		publisher:     dogfood.Publisher{Runner: dogfood.ExecCommandRunner{}},
		newIdempotencyDB: func(ledgerPath string) idempotencyStore {
			return fileIdempotencyStore{Path: markerPathForLedger(ledgerPath)}
		},
	}
}

func (d runtimeDeps) withDefaults() runtimeDeps {
	defaults := defaultRuntimeDeps()
	if d.stdout == nil {
		d.stdout = defaults.stdout
	}
	if d.stderr == nil {
		d.stderr = defaults.stderr
	}
	if d.getwd == nil {
		d.getwd = defaults.getwd
	}
	if d.readGitRemote == nil {
		d.readGitRemote = defaults.readGitRemote
	}
	if d.loadEvent == nil {
		d.loadEvent = defaults.loadEvent
	}
	if d.now == nil {
		d.now = defaults.now
	}
	if d.newLedger == nil {
		d.newLedger = defaults.newLedger
	}
	if d.publisher == nil {
		d.publisher = defaults.publisher
	}
	if d.newIdempotencyDB == nil {
		d.newIdempotencyDB = defaults.newIdempotencyDB
	}
	return d
}

func resolveExistingIssueURL(fp string, ledger ledgerStore, idempotency idempotencyStore) (string, error) {
	rec, ok, err := ledger.FindOpenByFingerprint(fp)
	if err != nil {
		return "", err
	}
	if ok {
		issueURL := strings.TrimSpace(rec.IssueURL)
		if issueURL == "" {
			return "", errors.New("existing open issue record has empty issue_url")
		}
		return issueURL, nil
	}

	issueURL, ok, err := idempotency.Get(fp)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	issueURL = strings.TrimSpace(issueURL)
	if issueURL == "" {
		return "", errors.New("idempotency marker has empty issue_url")
	}
	return issueURL, nil
}

type fileIdempotencyStore struct {
	Path string
}

func (s fileIdempotencyStore) Get(fingerprint string) (string, bool, error) {
	var (
		url string
		ok  bool
	)

	if err := s.withExclusiveLock(func() error {
		values, err := s.readAllUnlocked()
		if err != nil {
			return err
		}

		url, ok = values[strings.TrimSpace(fingerprint)]
		if !ok {
			return nil
		}
		url = strings.TrimSpace(url)
		if url == "" {
			ok = false
			return nil
		}
		return nil
	}); err != nil {
		return "", false, err
	}

	return url, ok, nil
}

func (s fileIdempotencyStore) Put(fingerprint, issueURL string) error {
	if strings.TrimSpace(s.Path) == "" {
		return errors.New("idempotency marker path is empty")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	issueURL = strings.TrimSpace(issueURL)
	if fingerprint == "" {
		return errors.New("fingerprint is required")
	}
	if issueURL == "" {
		return errors.New("issue_url is required")
	}

	return s.withExclusiveLock(func() error {
		values, err := s.readAllUnlocked()
		if err != nil {
			return err
		}
		values[fingerprint] = issueURL
		return s.writeAllUnlocked(values)
	})
}

func (s fileIdempotencyStore) readAllUnlocked() (map[string]string, error) {
	if strings.TrimSpace(s.Path) == "" {
		return nil, errors.New("idempotency marker path is empty")
	}

	raw, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read idempotency marker %q: %w", s.Path, err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]string{}, nil
	}

	var values map[string]string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode idempotency marker %q: %w", s.Path, err)
	}
	if values == nil {
		values = map[string]string{}
	}
	return values, nil
}

func (s fileIdempotencyStore) writeAllUnlocked(values map[string]string) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create idempotency marker dir %q: %w", dir, err)
	}

	raw, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("encode idempotency marker data: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, filepath.Base(s.Path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp idempotency marker file: %w", err)
	}
	tmpName := tmpFile.Name()

	cleanup := func(closeErr error) error {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
		return closeErr
	}

	if _, err := tmpFile.Write(raw); err != nil {
		return cleanup(fmt.Errorf("write temp idempotency marker file: %w", err))
	}
	if err := tmpFile.Close(); err != nil {
		return cleanup(fmt.Errorf("close temp idempotency marker file: %w", err))
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		return cleanup(fmt.Errorf("rename temp idempotency marker file: %w", err))
	}
	return nil
}

func (s fileIdempotencyStore) withExclusiveLock(fn func() error) error {
	unlock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

func (s fileIdempotencyStore) acquireLock() (func(), error) {
	if strings.TrimSpace(s.Path) == "" {
		return nil, errors.New("idempotency marker path is empty")
	}

	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create idempotency marker dir %q: %w", dir, err)
	}

	lockPath := s.Path + ".lock"
	deadline := time.Now().Add(idempotencyLockTimeout)
	for {
		lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = lock.Close()
			return func() {
				_ = os.Remove(lockPath)
			}, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create idempotency marker lock %q: %w", lockPath, err)
		}

		recovered, recoverErr := recoverStaleLock(lockPath, time.Now())
		if recoverErr != nil {
			return nil, recoverErr
		}
		if recovered {
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("acquire idempotency marker lock %q: timeout after %s", lockPath, idempotencyLockTimeout)
		}
		time.Sleep(idempotencyLockPoll)
	}
}

func recoverStaleLock(lockPath string, now time.Time) (bool, error) {
	info, err := os.Stat(lockPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat idempotency marker lock %q: %w", lockPath, err)
	}

	if now.Sub(info.ModTime()) < idempotencyStaleLock {
		return false, nil
	}

	if err := os.Remove(lockPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("remove stale idempotency marker lock %q: %w", lockPath, err)
	}

	return true, nil
}

func markerPathForLedger(ledgerPath string) string {
	path := strings.TrimSpace(ledgerPath)
	if path == "" {
		path = defaultLedgerPath
	}
	return path + ".idempotency.json"
}

func loadEvent(path string) (dogfood.Event, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return dogfood.Event{}, fmt.Errorf("read %q: %w", path, err)
	}

	var event dogfood.Event
	if err := json.Unmarshal(raw, &event); err != nil {
		return dogfood.Event{}, fmt.Errorf("decode %q: %w", path, err)
	}
	return event, nil
}

func readGitRemote(cwd string) string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func issueTitle(event dogfood.Event) string {
	eventType := strings.TrimSpace(string(event.EventType))
	if eventType == "" {
		eventType = "unknown_event"
	}

	summary := strings.TrimSpace(event.ErrorSummary)
	if summary == "" {
		return "dogfood: " + eventType
	}
	return "dogfood: " + eventType + " - " + summary
}

func issueBody(event dogfood.Event, route dogfood.RouteResult, fingerprint string) string {
	lines := []string{
		"Automated dogfood feedback event.",
		"",
		"- event_id: " + strings.TrimSpace(event.EventID),
		"- event_type: " + strings.TrimSpace(string(event.EventType)),
		"- signal_source: " + strings.TrimSpace(event.SignalSource),
		"- timestamp: " + event.Timestamp.UTC().Format(time.RFC3339),
		"- repo_route: " + strings.TrimSpace(route.Repo),
		fmt.Sprintf("- repo_confidence: %.2f", route.Confidence),
		"- route_reason: " + strings.TrimSpace(route.Reason),
		"- fingerprint: " + strings.TrimSpace(fingerprint),
	}
	if summary := strings.TrimSpace(event.ErrorSummary); summary != "" {
		lines = append(lines, "", "Error summary:", summary)
	}
	if len(event.EvidencePaths) > 0 {
		lines = append(lines, "", "Evidence paths:")
		for _, path := range event.EvidencePaths {
			lines = append(lines, "- "+path)
		}
	}
	return strings.Join(lines, "\n")
}
