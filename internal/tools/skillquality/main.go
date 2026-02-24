package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type roleContext struct {
	RepoRoot string `json:"repo_root"`
}

type judgerFinding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

type skillQualityJudgerOutput struct {
	SchemaVersion string          `json:"schema_version"`
	ExtraFindings []judgerFinding `json:"extra_findings"`
	Notes         string          `json:"notes"`
}

func main() {
	contextPath := flag.String("context", "", "path to role context json")
	repoRoot := flag.String("repo-root", "", "repository root for checks")
	flag.Parse()

	if *contextPath == "" {
		fmt.Fprintln(os.Stderr, "context is required")
		os.Exit(2)
	}
	if *repoRoot == "" {
		ctx, err := loadRoleContext(*contextPath)
		if err == nil && strings.TrimSpace(ctx.RepoRoot) != "" {
			repoRoot = &ctx.RepoRoot
		}
		if *repoRoot == "" {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			repoRoot = &cwd
		}
	}

	findings := checkSkillFiles(*repoRoot)
	notes := "skill quality checks passed"
	if len(findings) > 0 {
		notes = fmt.Sprintf("skill quality checks found %d issue(s)", len(findings))
	}

	out := skillQualityJudgerOutput{
		SchemaVersion: "v1",
		ExtraFindings: findings,
		Notes:         notes,
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
}

func checkSkillFiles(repoRoot string) []judgerFinding {
	findings := make([]judgerFinding, 0)
	skillPath := filepath.Join(repoRoot, "skills", "agentcli-go", "SKILL.md")
	openaiPath := filepath.Join(repoRoot, "skills", "agentcli-go", "agents", "openai.yaml")

	if err := checkSkillDoc(skillPath, &findings); err != nil {
		findings = append(findings, judgerFinding{
			Code:     "skill_file_unreadable",
			Severity: "high",
			Message:  err.Error(),
			Source:   "skills/agentcli-go/SKILL.md",
		})
	}
	if err := checkOpenAIPrompt(openaiPath, &findings); err != nil {
		findings = append(findings, judgerFinding{
			Code:     "skill_file_unreadable",
			Severity: "high",
			Message:  err.Error(),
			Source:   "skills/agentcli-go/agents/openai.yaml",
		})
	}

	return findings
}

func checkSkillDoc(path string, findings *[]judgerFinding) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read skill markdown: %w", err)
	}
	content := string(raw)
	ensureFrontmatter(content, path, findings)
	if hasPlaceholder(content) {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_content_issue",
			Severity: "medium",
			Message:  "skill markdown likely includes placeholder text",
			Source:   path,
		})
	}
	return nil
}

func ensureFrontmatter(text, path string, findings *[]judgerFinding) {
	fm, ok := extractFrontmatter(text)
	if !ok {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_frontmatter_missing",
			Severity: "high",
			Message:  "skill markdown is missing YAML frontmatter",
			Source:   path,
		})
		return
	}
	name := strings.TrimSpace(fm["name"])
	desc := strings.TrimSpace(fm["description"])
	if name == "" {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_frontmatter_missing",
			Severity: "medium",
			Message:  "frontmatter missing required `name`",
			Source:   path,
		})
	}
	if desc == "" {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_frontmatter_missing",
			Severity: "medium",
			Message:  "frontmatter missing required `description`",
			Source:   path,
		})
	}
	if !strings.Contains(strings.ToLower(text), "## use this when") {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_content_issue",
			Severity: "low",
			Message:  "skill markdown should include a '## Use this when' section",
			Source:   path,
		})
	}
}

func checkOpenAIPrompt(path string, findings *[]judgerFinding) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read openai prompt config: %w", err)
	}
	values := parseSimpleYAML(string(raw))
	displayName := strings.TrimSpace(values["display_name"])
	prompt := strings.TrimSpace(values["default_prompt"])
	shortDesc := strings.TrimSpace(values["short_description"])

	if displayName == "" {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_openai_config_issue",
			Severity: "high",
			Message:  "agents/openai.yaml missing `display_name`",
			Source:   path,
		})
	}
	if prompt == "" {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_openai_config_issue",
			Severity: "high",
			Message:  "agents/openai.yaml missing `default_prompt`",
			Source:   path,
		})
	}
	if shortDesc == "" {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_openai_config_issue",
			Severity: "medium",
			Message:  "agents/openai.yaml missing optional `short_description`",
			Source:   path,
		})
	}
	if hasPlaceholder(string(raw)) {
		*findings = append(*findings, judgerFinding{
			Code:     "skill_openai_config_issue",
			Severity: "medium",
			Message:  "agents/openai.yaml likely includes placeholder text",
			Source:   path,
		})
	}
	return nil
}

func loadRoleContext(path string) (roleContext, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return roleContext{}, fmt.Errorf("read context: %w", err)
	}
	var ctx roleContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return roleContext{}, fmt.Errorf("parse context: %w", err)
	}
	return ctx, nil
}

func extractFrontmatter(text string) (map[string]string, bool) {
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return nil, false
	}
	fm := make(map[string]string)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "---" {
			return fm, true
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		fm[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
	}
	return nil, false
}

func parseSimpleYAML(text string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), "\"'")
	}
	return out
}

func hasPlaceholder(s string) bool {
	text := strings.ToLower(s)
	for _, marker := range []string{"todo", "tbd", "placeholder", "fixme"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
