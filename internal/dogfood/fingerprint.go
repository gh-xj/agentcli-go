package dogfood

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

func Fingerprint(e Event) string {
	payload := struct {
		RepoGuess     string    `json:"repo_guess"`
		EventType     EventType `json:"event_type"`
		SignalSource  string    `json:"signal_source"`
		ErrorSummary  string    `json:"error_summary"`
		EvidencePaths []string  `json:"evidence_paths"`
	}{
		RepoGuess:     strings.TrimSpace(e.RepoGuess),
		EventType:     e.EventType,
		SignalSource:  strings.TrimSpace(e.SignalSource),
		ErrorSummary:  normalizeErrorSummary(e.ErrorSummary),
		EvidencePaths: canonicalEvidencePaths(e.EvidencePaths),
	}

	base, err := json.Marshal(payload)
	if err != nil {
		base = []byte("fingerprint-payload-serialization-error")
	}

	sum := sha256.Sum256(base)
	return hex.EncodeToString(sum[:12])
}

func normalizeErrorSummary(summary string) string {
	return strings.ToLower(strings.Join(strings.Fields(summary), " "))
}

func canonicalEvidencePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	evidence := append([]string(nil), paths...)
	sort.Strings(evidence)

	dedup := evidence[:0]
	for i, path := range evidence {
		if i == 0 || path != evidence[i-1] {
			dedup = append(dedup, path)
		}
	}

	return append([]string(nil), dedup...)
}
