package dogfood

type Engine struct {
	MinPublishConfidence float64
}

func (e Engine) Decide(in DecisionInput) Decision {
	if in.RepoConfidence < e.minPublishConfidence() {
		return Decision{Action: ActionPendingReview, Reason: "low_confidence_route"}
	}
	if in.HasOpenIssue {
		return Decision{Action: ActionAppendComment, Reason: "dedupe_open_issue"}
	}
	return Decision{Action: ActionCreateIssue, Reason: "new_fingerprint"}
}

func (e Engine) minPublishConfidence() float64 {
	if e.MinPublishConfidence <= 0 || e.MinPublishConfidence > 1 {
		return DefaultMinConfidence
	}
	return e.MinPublishConfidence
}
