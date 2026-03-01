package dogfood

import "testing"

func TestEngineMarksPendingWhenConfidenceBelowThreshold(t *testing.T) {
	eng := Engine{MinPublishConfidence: 0.8}

	dec := eng.Decide(DecisionInput{
		RepoConfidence: 0.4,
		Fingerprint:    "fp-1",
	})
	if dec.Action != ActionPendingReview {
		t.Fatalf("expected action %q, got %q", ActionPendingReview, dec.Action)
	}
	if dec.Reason != "low_confidence_route" {
		t.Fatalf("expected low confidence reason, got %q", dec.Reason)
	}
}

func TestEngineAppendsCommentWhenOpenIssueExists(t *testing.T) {
	eng := Engine{MinPublishConfidence: 0.8}

	dec := eng.Decide(DecisionInput{
		RepoConfidence: 0.9,
		HasOpenIssue:   true,
		Fingerprint:    "fp-1",
	})
	if dec.Action != ActionAppendComment {
		t.Fatalf("expected action %q, got %q", ActionAppendComment, dec.Action)
	}
	if dec.Reason != "dedupe_open_issue" {
		t.Fatalf("expected dedupe reason, got %q", dec.Reason)
	}
}

func TestEngineCreatesIssueWhenRouteConfidentAndNoOpenIssue(t *testing.T) {
	eng := Engine{MinPublishConfidence: 0.8}

	dec := eng.Decide(DecisionInput{
		RepoConfidence: 0.9,
		HasOpenIssue:   false,
		Fingerprint:    "fp-1",
	})
	if dec.Action != ActionCreateIssue {
		t.Fatalf("expected action %q, got %q", ActionCreateIssue, dec.Action)
	}
	if dec.Reason != "new_fingerprint" {
		t.Fatalf("expected new fingerprint reason, got %q", dec.Reason)
	}
}
