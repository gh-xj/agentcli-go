package harnessloop

import (
	"os"
	"path/filepath"
	"strings"
)

func CheckOnboardingInstallReadiness(repoRoot string) []Finding {
	findings := make([]Finding, 0)
	readmePath := filepath.Join(repoRoot, "README.md")
	promptPath := filepath.Join(repoRoot, "prompts", "agentcli-onboarding.prompt.md")

	readme, _ := os.ReadFile(readmePath)
	prompt, _ := os.ReadFile(promptPath)
	readmeText := strings.ToLower(string(readme))
	promptText := strings.ToLower(string(prompt))

	if !containsInstallStep(promptText) {
		findings = append(findings, Finding{
			Code:     "onboarding_install_missing",
			Severity: "high",
			Message:  "onboarding prompt is missing explicit agentops install step",
			Source:   "prompts/agentcli-onboarding.prompt.md",
		})
	}
	if !containsInstallVerificationStep(promptText) {
		findings = append(findings, Finding{
			Code:     "onboarding_install_verify_missing",
			Severity: "medium",
			Message:  "onboarding prompt is missing install verification step",
			Source:   "prompts/agentcli-onboarding.prompt.md",
		})
	}
	if strings.Contains(readmeText, "ai prompt starter") && !containsInstallStep(readmeText) {
		findings = append(findings, Finding{
			Code:     "onboarding_install_missing",
			Severity: "high",
			Message:  "README AI prompt starter is missing explicit install step",
			Source:   "README.md",
		})
	}
	return findings
}

func containsInstallStep(s string) bool {
	return strings.Contains(s, "go install github.com/gh-xj/agentops/cmd/agentops@") ||
		(strings.Contains(s, "brew") && strings.Contains(s, "install agentops"))
}

func containsInstallVerificationStep(s string) bool {
	return strings.Contains(s, "which agentops") ||
		(strings.Contains(s, "agentops --version") && strings.Contains(s, "agentops --help"))
}
