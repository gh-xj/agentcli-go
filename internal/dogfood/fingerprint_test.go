package dogfood

import "testing"

func TestFingerprintStableAcrossEvidenceOrder(t *testing.T) {
	a := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"b.log", "a.log"},
	}
	b := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"a.log", "b.log"},
	}

	if Fingerprint(a) != Fingerprint(b) {
		t.Fatalf("fingerprint mismatch")
	}
}

func TestFingerprintDistinguishesAmbiguousEvidenceDelimiters(t *testing.T) {
	a := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"a,b", "c"},
	}
	b := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"a", "b,c"},
	}

	if Fingerprint(a) == Fingerprint(b) {
		t.Fatalf("fingerprint should differ for distinct evidence payloads")
	}
}

func TestFingerprintIgnoresDuplicateEvidencePaths(t *testing.T) {
	a := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"a.log", "a.log", "b.log"},
	}
	b := Event{
		RepoGuess:     "org/repo",
		EventType:     EventTypeRuntimeError,
		SignalSource:  "local",
		ErrorSummary:  "panic: boom",
		EvidencePaths: []string{"b.log", "a.log"},
	}

	if Fingerprint(a) != Fingerprint(b) {
		t.Fatalf("fingerprint should ignore duplicate evidence paths")
	}
}
