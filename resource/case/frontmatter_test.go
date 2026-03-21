package caseresource

import (
	"strings"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	content := "---\ntype: intake\nstatus: open\nclaimed_by: agent-1\ncreated: \"20250101\"\n---\n# Title\n\nBody text here.\n"

	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Type != "intake" {
		t.Errorf("Type = %q, want %q", fm.Type, "intake")
	}
	if fm.Status != "open" {
		t.Errorf("Status = %q, want %q", fm.Status, "open")
	}
	if fm.ClaimedBy != "agent-1" {
		t.Errorf("ClaimedBy = %q, want %q", fm.ClaimedBy, "agent-1")
	}
	if fm.Created != "20250101" {
		t.Errorf("Created = %q, want %q", fm.Created, "20250101")
	}
	if !strings.HasPrefix(body, "# Title") {
		t.Errorf("body should start with '# Title', got %q", body)
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	content := "# Just a heading\n\nNo frontmatter here.\n"
	_, _, err := ParseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error when no frontmatter present")
	}
}

func TestParseFrontmatterUnterminated(t *testing.T) {
	content := "---\ntype: intake\nstatus: open\n"
	_, _, err := ParseFrontmatter(content)
	if err == nil {
		t.Fatal("expected error for unterminated frontmatter")
	}
}

func TestParseFrontmatterEmptyFields(t *testing.T) {
	content := "---\ntype: \"\"\nstatus: \"\"\nclaimed_by: \"\"\ncreated: \"\"\n---\nBody\n"
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Type != "" {
		t.Errorf("Type = %q, want empty", fm.Type)
	}
}

func TestRenderFrontmatter(t *testing.T) {
	fm := Frontmatter{
		Type:      "intake",
		Status:    "open",
		ClaimedBy: "agent-1",
		Created:   "20250101",
	}

	rendered := RenderFrontmatter(fm)

	if !strings.HasPrefix(rendered, "---\n") {
		t.Error("should start with ---")
	}
	if !strings.HasSuffix(rendered, "---\n") {
		t.Error("should end with ---")
	}
	if !strings.Contains(rendered, "type: intake\n") {
		t.Error("should contain type field")
	}
	if !strings.Contains(rendered, "status: open\n") {
		t.Error("should contain status field")
	}
	if !strings.Contains(rendered, "claimed_by: agent-1\n") {
		t.Error("should contain claimed_by field")
	}
	if !strings.Contains(rendered, `created: "20250101"`) {
		t.Error("should contain quoted created field")
	}
}

func TestRenderThenParse(t *testing.T) {
	original := Frontmatter{
		Type:      "bug",
		Status:    "in_progress",
		ClaimedBy: "dev-2",
		Created:   "20250315",
	}

	rendered := RenderFrontmatter(original)
	parsed, body, err := ParseFrontmatter(rendered + "# Body\n")
	if err != nil {
		t.Fatalf("parse after render: %v", err)
	}
	if parsed.Type != original.Type {
		t.Errorf("Type roundtrip: got %q, want %q", parsed.Type, original.Type)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status roundtrip: got %q, want %q", parsed.Status, original.Status)
	}
	if parsed.ClaimedBy != original.ClaimedBy {
		t.Errorf("ClaimedBy roundtrip: got %q, want %q", parsed.ClaimedBy, original.ClaimedBy)
	}
	if parsed.Created != original.Created {
		t.Errorf("Created roundtrip: got %q, want %q", parsed.Created, original.Created)
	}
	if body != "# Body\n" {
		t.Errorf("body = %q, want %q", body, "# Body\n")
	}
}
