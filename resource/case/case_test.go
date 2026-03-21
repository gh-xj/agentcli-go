package caseresource

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/strategy"
)

// setupTestProject creates a temp dir with .agentops/ bootstrapped and storage set to in-repo.
func setupTestProject(t *testing.T) (string, *strategy.Strategy) {
	t.Helper()
	tmp := t.TempDir()

	if err := strategy.Bootstrap(tmp); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Override storage to in-repo
	storageYAML := filepath.Join(tmp, ".agentops", "storage.yaml")
	if err := os.WriteFile(storageYAML, []byte("backend: in-repo\n"), 0o644); err != nil {
		t.Fatalf("write storage.yaml: %v", err)
	}

	strat, err := strategy.Discover(tmp)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	return tmp, strat
}

func testCtx() *agentcli.AppContext {
	return agentcli.NewAppContext(context.Background())
}

func TestCaseResourceSchema(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)

	schema := cr.Schema()
	if schema.Kind != "case" {
		t.Errorf("Kind = %q, want %q", schema.Kind, "case")
	}
	if len(schema.Fields) == 0 {
		t.Error("expected fields in schema")
	}
	if len(schema.Statuses) == 0 {
		t.Error("expected statuses in schema")
	}

	// Check that known fields exist
	fieldNames := make(map[string]bool)
	for _, f := range schema.Fields {
		fieldNames[f.Name] = true
	}
	for _, name := range []string{"id", "type", "status", "claimed_by", "created"} {
		if !fieldNames[name] {
			t.Errorf("missing field %q in schema", name)
		}
	}
}

func TestCaseResourceCreate(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	rec, err := cr.Create(ctx, "my-test-case", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Kind != "case" {
		t.Errorf("Kind = %q, want %q", rec.Kind, "case")
	}
	if !strings.HasPrefix(rec.ID, "CASE-") {
		t.Errorf("ID = %q, should start with CASE-", rec.ID)
	}
	if !strings.HasSuffix(rec.ID, "my-test-case") {
		t.Errorf("ID = %q, should end with 'my-test-case'", rec.ID)
	}
	if rec.Fields["status"] != "open" {
		t.Errorf("status = %v, want 'open'", rec.Fields["status"])
	}

	// Verify case.md exists on disk
	caseMDPath := filepath.Join(strat.Root, "cases", rec.ID, "case.md")
	if _, err := os.Stat(caseMDPath); err != nil {
		t.Errorf("case.md not found at %s: %v", caseMDPath, err)
	}
}

func TestCaseResourceCreateInvalidSlug(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	tests := []string{
		"UPPERCASE",
		"-starts-with-dash",
		"has spaces",
		"has_underscore",
		"",
	}

	for _, slug := range tests {
		t.Run(slug, func(t *testing.T) {
			_, err := cr.Create(ctx, slug, nil)
			if err == nil {
				t.Errorf("expected error for invalid slug %q", slug)
			}
		})
	}
}

func TestCaseResourceCreateCollision(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	rec1, err := cr.Create(ctx, "collision", nil)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	rec2, err := cr.Create(ctx, "collision", nil)
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}

	if rec1.ID == rec2.ID {
		t.Errorf("expected different IDs for collision, got %q both times", rec1.ID)
	}
	if !strings.HasSuffix(rec2.ID, "-02") {
		t.Errorf("second case should end with -02, got %q", rec2.ID)
	}
}

func TestCaseResourceList(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	// Create some cases
	_, err := cr.Create(ctx, "alpha", nil)
	if err != nil {
		t.Fatalf("create alpha: %v", err)
	}
	_, err = cr.Create(ctx, "beta", nil)
	if err != nil {
		t.Fatalf("create beta: %v", err)
	}

	// List all
	records, err := cr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestCaseResourceListFilterByStatus(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	_, err := cr.Create(ctx, "open-case", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Filter by exact status
	records, err := cr.List(ctx, map[string]string{"status": "open"})
	if err != nil {
		t.Fatalf("List with status filter: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record for status=open, got %d", len(records))
	}

	// Filter by status group
	records, err = cr.List(ctx, map[string]string{"status": "active"})
	if err != nil {
		t.Fatalf("List with group filter: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record for status=active, got %d", len(records))
	}

	// Filter by non-matching status
	records, err = cr.List(ctx, map[string]string{"status": "resolved"})
	if err != nil {
		t.Fatalf("List with resolved filter: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records for status=resolved, got %d", len(records))
	}
}

func TestCaseResourceGet(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	created, err := cr.Create(ctx, "get-test", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := cr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q, want %q", got.ID, created.ID)
	}
	if got.Fields["status"] != "open" {
		t.Errorf("status = %v, want 'open'", got.Fields["status"])
	}
}

func TestCaseResourceGetNotFound(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	_, err := cr.Get(ctx, "CASE-99999999-nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent case")
	}
}

func TestCaseResourceValidate(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	created, err := cr.Create(ctx, "validate-test", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	report, err := cr.Validate(ctx, created.ID)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !report.OK {
		t.Errorf("expected OK for valid case, got findings: %v", report.Findings)
	}
}

func TestCaseResourceValidateMissingFields(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	// Create a case then corrupt its frontmatter
	created, err := cr.Create(ctx, "corrupt-test", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	caseMDPath := filepath.Join(strat.Root, "cases", created.ID, "case.md")
	if err := os.WriteFile(caseMDPath, []byte("---\nstatus: open\n---\n# No type or created\n"), 0o644); err != nil {
		t.Fatalf("write corrupted case.md: %v", err)
	}

	report, err := cr.Validate(ctx, created.ID)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if report.OK {
		t.Error("expected NOT OK for case missing type and created")
	}
	if len(report.Findings) < 2 {
		t.Errorf("expected at least 2 findings, got %d", len(report.Findings))
	}
}

func TestCaseResourceTransition(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	created, err := cr.Create(ctx, "transition-test", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Fields["status"] != "open" {
		t.Fatalf("initial status = %v, want 'open'", created.Fields["status"])
	}

	// Transition open -> in_progress
	updated, err := cr.Transition(ctx, created.ID, "start")
	if err != nil {
		t.Fatalf("Transition start: %v", err)
	}
	if updated.Fields["status"] != "in_progress" {
		t.Errorf("status after start = %v, want 'in_progress'", updated.Fields["status"])
	}

	// Verify persisted on disk
	got, err := cr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after transition: %v", err)
	}
	if got.Fields["status"] != "in_progress" {
		t.Errorf("persisted status = %v, want 'in_progress'", got.Fields["status"])
	}
}

func TestCaseResourceTransitionInvalid(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	created, err := cr.Create(ctx, "bad-transition", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// open -> resolve is not valid (must go through in_progress first)
	_, err = cr.Transition(ctx, created.ID, "resolve")
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
}

func TestCaseResourceTransitionCategoryMove(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	created, err := cr.Create(ctx, "category-move", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// open -> in_progress (stays in active)
	_, err = cr.Transition(ctx, created.ID, "start")
	if err != nil {
		t.Fatalf("transition start: %v", err)
	}

	// in_progress -> resolved (active -> completed)
	_, err = cr.Transition(ctx, created.ID, "resolve")
	if err != nil {
		t.Fatalf("transition resolve: %v", err)
	}

	// Verify the case is still accessible
	got, err := cr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get after category move: %v", err)
	}
	if got.Fields["status"] != "resolved" {
		t.Errorf("status = %v, want 'resolved'", got.Fields["status"])
	}
}

func TestCaseResourceNilStrategy(t *testing.T) {
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, nil)

	ctx := testCtx()
	_, err := cr.Create(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error when strategy is nil")
	}
}

func TestCaseResourceSeparateRepoStorage(t *testing.T) {
	tmp := t.TempDir()

	if err := strategy.Bootstrap(tmp); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	// Leave storage as separate-repo (default)
	strat, err := strategy.Discover(tmp)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}

	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	// Create should work — it'll create the cases dir in the separate repo path
	rec, err := cr.Create(ctx, "separate-test", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify cases dir is under the expected separate repo location
	expectedBase := filepath.Base(tmp) + "-cases"
	expectedCasesDir := filepath.Join(filepath.Dir(tmp), expectedBase, "cases")
	caseMDPath := filepath.Join(expectedCasesDir, rec.ID, "case.md")
	if _, err := os.Stat(caseMDPath); err != nil {
		t.Errorf("case.md not found at expected separate-repo path %s: %v", caseMDPath, err)
	}
}

func TestCaseResourceListFilterBySlot(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	// Create a case
	created, err := cr.Create(ctx, "slot-filter", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Manually set claimed_by in case.md
	caseMDPath := filepath.Join(strat.Root, "cases", created.ID, "case.md")
	data, err := os.ReadFile(caseMDPath)
	if err != nil {
		t.Fatalf("read case.md: %v", err)
	}
	content := strings.ReplaceAll(string(data), "claimed_by: none", "claimed_by: agent-1")
	if err := os.WriteFile(caseMDPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write case.md: %v", err)
	}

	// Filter by slot (claimed_by)
	records, err := cr.List(ctx, map[string]string{"slot": "agent-1"})
	if err != nil {
		t.Fatalf("List with slot filter: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record for slot=agent-1, got %d", len(records))
	}

	// Filter by different slot
	records, err = cr.List(ctx, map[string]string{"slot": "other"})
	if err != nil {
		t.Fatalf("List with other slot filter: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records for slot=other, got %d", len(records))
	}
}

func TestCaseResourceCreateSlugMaxLength(t *testing.T) {
	_, strat := setupTestProject(t)
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	cr := New(fs, exec, strat)
	ctx := testCtx()

	longSlug := strings.Repeat("a", 129)
	_, err := cr.Create(ctx, longSlug, nil)
	if err == nil {
		t.Fatal("expected error for slug exceeding 128 chars")
	}
}
