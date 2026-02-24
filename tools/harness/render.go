package harness

import (
	"encoding/json"
	"fmt"
	"strings"
)

func RenderSummary(summary CommandSummary, format string, noColor bool) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		return renderText(summary, noColor), nil
	case "json":
		raw, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return "", err
		}
		return string(raw) + "\n", nil
	case "ndjson":
		raw, err := json.Marshal(summary)
		if err != nil {
			return "", err
		}
		return string(raw) + "\n", nil
	default:
		return "", NewFailure(CodeUsage, "invalid output format", "use --format text|json|ndjson", false)
	}
}

func renderText(summary CommandSummary, _ bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("command: %s\n", summary.Command))
	b.WriteString(fmt.Sprintf("status: %s\n", summary.Status))
	b.WriteString(fmt.Sprintf("duration_ms: %d\n", summary.DurationMs))

	if len(summary.Checks) > 0 {
		b.WriteString("checks:\n")
		for _, check := range summary.Checks {
			line := fmt.Sprintf("- [%s] %s", check.Status, check.Name)
			if strings.TrimSpace(check.Details) != "" {
				line += ": " + check.Details
			}
			b.WriteString(line + "\n")
		}
	}

	if len(summary.Failures) > 0 {
		b.WriteString("failures:\n")
		for _, failure := range summary.Failures {
			line := fmt.Sprintf("- [%s] %s", failure.Code, failure.Message)
			if strings.TrimSpace(failure.Hint) != "" {
				line += fmt.Sprintf(" (hint: %s)", failure.Hint)
			}
			b.WriteString(line + "\n")
		}
	}

	if len(summary.Artifacts) > 0 {
		b.WriteString("artifacts:\n")
		for _, artifact := range summary.Artifacts {
			label := strings.TrimSpace(artifact.Name)
			if label == "" {
				label = "artifact"
			}
			if strings.TrimSpace(artifact.Kind) != "" {
				label = fmt.Sprintf("%s(%s)", label, artifact.Kind)
			}
			if strings.TrimSpace(artifact.Path) != "" {
				b.WriteString(fmt.Sprintf("- %s: %s\n", label, artifact.Path))
				continue
			}
			b.WriteString(fmt.Sprintf("- %s\n", label))
		}
	}

	return b.String()
}
