package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gh-xj/agentcli-go/internal/dogfood"
)

const defaultLedgerPath = ".docs/dogfood/ledger.json"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("dogfoodfeedback", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	eventPath := fs.String("event", "", "path to dogfood event json")
	ledgerPath := fs.String("ledger", defaultLedgerPath, "path to dogfood ledger json")
	overrideRepo := fs.String("repo", "", "override target repo (owner/name)")
	dryRun := fs.Bool("dry-run", false, "print decision without publishing")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*eventPath) == "" {
		fmt.Fprintln(os.Stderr, "--event is required")
		fs.Usage()
		return 2
	}

	event, err := loadEvent(*eventPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load event: %v\n", err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve cwd: %v\n", err)
		return 1
	}

	route := dogfood.Router{}.Resolve(dogfood.RouteInput{
		OverrideRepo: strings.TrimSpace(*overrideRepo),
		CWD:          cwd,
		GitRemote:    readGitRemote(cwd),
	})
	if strings.TrimSpace(event.RepoGuess) == "" {
		event.RepoGuess = route.Repo
	}

	fp := dogfood.Fingerprint(event)
	ledger := dogfood.NewLedger(*ledgerPath)

	existing, hasOpen, err := ledger.FindOpenByFingerprint(fp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lookup open ledger record: %v\n", err)
		return 1
	}

	decision := dogfood.Engine{MinPublishConfidence: dogfood.DefaultMinConfidence}.Decide(dogfood.DecisionInput{
		RepoConfidence: route.Confidence,
		HasOpenIssue:   hasOpen,
		Fingerprint:    fp,
	})

	if *dryRun {
		existingIssue := ""
		if hasOpen {
			existingIssue = existing.IssueURL
		}
		fmt.Fprintf(os.Stdout, "decision=%s reason=%s repo=%s confidence=%.2f fingerprint=%s existing_issue=%s\n", decision.Action, decision.Reason, route.Repo, route.Confidence, fp, existingIssue)
		return 0
	}

	now := time.Now().UTC()
	if decision.Action == dogfood.ActionPendingReview {
		if err := ledger.Append(dogfood.LedgerRecord{
			SchemaVersion: dogfood.LedgerSchemaVersionV1,
			EventID:       strings.TrimSpace(event.EventID),
			Fingerprint:   fp,
			Status:        string(dogfood.ActionPendingReview),
			CreatedAt:     now,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "append pending ledger record: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stdout, "decision=%s reason=%s fingerprint=%s\n", decision.Action, decision.Reason, fp)
		return 0
	}

	publishInput := dogfood.PublishInput{
		Repo:  route.Repo,
		Title: issueTitle(event),
		Body:  issueBody(event, route, fp),
	}
	if decision.Action == dogfood.ActionAppendComment {
		publishInput.ExistingIssueURL = strings.TrimSpace(existing.IssueURL)
	}

	issueURL, publishAction, err := (dogfood.Publisher{Runner: dogfood.ExecCommandRunner{}}).Publish(publishInput)
	if err != nil {
		_ = ledger.Append(dogfood.LedgerRecord{
			SchemaVersion: dogfood.LedgerSchemaVersionV1,
			EventID:       strings.TrimSpace(event.EventID),
			Fingerprint:   fp,
			Status:        string(dogfood.ActionQueueRetry),
			CreatedAt:     now,
		})
		fmt.Fprintf(os.Stderr, "publish feedback: %v\n", err)
		return 1
	}

	if err := ledger.Append(dogfood.LedgerRecord{
		SchemaVersion: dogfood.LedgerSchemaVersionV1,
		EventID:       strings.TrimSpace(event.EventID),
		Fingerprint:   fp,
		IssueURL:      issueURL,
		Status:        dogfood.LedgerStatusOpen,
		CreatedAt:     now,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "append open ledger record: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "action=%s issue=%s fingerprint=%s\n", publishAction, issueURL, fp)
	return 0
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
