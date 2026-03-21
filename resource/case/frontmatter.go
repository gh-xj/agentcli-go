package caseresource

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter represents YAML frontmatter in case.md files.
type Frontmatter struct {
	Type      string `yaml:"type"`
	Status    string `yaml:"status"`
	ClaimedBy string `yaml:"claimed_by"`
	Created   string `yaml:"created"`
}

// ParseFrontmatter extracts YAML frontmatter from case.md content.
// Returns the parsed frontmatter and the remaining body content.
func ParseFrontmatter(content string) (Frontmatter, string, error) {
	var fm Frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return fm, content, fmt.Errorf("no YAML frontmatter found")
	}
	rest := content[4:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx < 0 {
		return fm, content, fmt.Errorf("unterminated YAML frontmatter")
	}
	yamlBlock := rest[:endIdx]
	body := rest[endIdx+4:] // skip "\n---"
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	}
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return fm, content, fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}

// RenderFrontmatter renders a Frontmatter as a YAML frontmatter block.
func RenderFrontmatter(fm Frontmatter) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("type: " + fm.Type + "\n")
	b.WriteString("status: " + fm.Status + "\n")
	b.WriteString("claimed_by: " + fm.ClaimedBy + "\n")
	// Quote created to prevent YAML date parsing.
	b.WriteString("created: \"" + fm.Created + "\"\n")
	b.WriteString("---\n")
	return b.String()
}
