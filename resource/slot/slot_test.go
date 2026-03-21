package slotresource

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/resource"
)

// setupGitRepo creates a temporary git repo with an initial commit.
// Returns the repo path.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = tmp
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("setup %v: %s: %v", args, out, err)
		}
	}
	return tmp
}

// newTestResource creates a SlotResource wired to real dal implementations
// with the projectDir override set.
func newTestResource(t *testing.T, projectDir string) (*SlotResource, *agentcli.AppContext) {
	t.Helper()
	sr := New(&realFS{}, &realExec{})
	ctx := agentcli.NewAppContext(context.Background())
	ctx.Values["project_dir"] = projectDir
	return sr, ctx
}

// realFS implements dal.FileSystem using the real OS.
type realFS struct{}

func (f *realFS) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *realFS) EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func (f *realFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *realFS) WriteFile(path string, data []byte, perm int) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

func (f *realFS) ReadDir(path string) ([]dal.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var result []dal.DirEntry
	for _, e := range entries {
		result = append(result, dal.DirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return result, nil
}

func (f *realFS) BaseName(path string) string {
	return filepath.Base(path)
}

// realExec implements dal.Executor using real os/exec.
type realExec struct{}

func (e *realExec) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *realExec) RunInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (e *realExec) RunOsascript(script string) string { return "" }
func (e *realExec) Which(cmd string) bool             { return false }

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ resource.Resource = (*SlotResource)(nil)
	var _ resource.Deleter = (*SlotResource)(nil)
	var _ resource.Syncer = (*SlotResource)(nil)
}

// --- Schema ---

func TestSchema(t *testing.T) {
	sr := New(&realFS{}, &realExec{})
	s := sr.Schema()
	if s.Kind != "slot" {
		t.Errorf("Schema().Kind = %q, want %q", s.Kind, "slot")
	}
	if len(s.Fields) == 0 {
		t.Error("Schema().Fields is empty")
	}
}

// --- Create ---

func TestSlotCreate(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	rec, err := sr.Create(ctx, "feat-login", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if rec.Kind != "slot" {
		t.Errorf("record Kind = %q, want %q", rec.Kind, "slot")
	}
	if rec.ID != "feat-login" {
		t.Errorf("record ID = %q, want %q", rec.ID, "feat-login")
	}

	// Check .slot marker exists at worktree path
	wtPath, ok := rec.Fields["path"].(string)
	if !ok || wtPath == "" {
		t.Fatal("record missing path field")
	}
	slotFile := filepath.Join(wtPath, ".slot")
	data, err := os.ReadFile(slotFile)
	if err != nil {
		t.Fatalf(".slot marker not found: %v", err)
	}
	if string(data) != "feat-login" {
		t.Errorf(".slot content = %q, want %q", string(data), "feat-login")
	}

	// Check branch field
	branch, ok := rec.Fields["branch"].(string)
	if !ok || branch == "" {
		t.Fatal("record missing branch field")
	}
}

// --- Name validation ---

func TestSlotNameValidation(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	cases := []struct {
		name    string
		wantErr bool
	}{
		{"good-name", false},
		{"a", false},
		{"abc123", false},
		{"a-b-c", false},
		{"", true},
		{"123abc", true},
		{"-bad", true},
		{"UPPER", true},
		{"has space", true},
		{"has_underscore", true},
	}
	for _, tc := range cases {
		_, err := sr.Create(ctx, tc.name, nil)
		if tc.wantErr && err == nil {
			t.Errorf("Create(%q): expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("Create(%q): unexpected error: %v", tc.name, err)
		}
	}
}

// --- List ---

func TestSlotList(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	// Create two slots
	_, err := sr.Create(ctx, "alpha", nil)
	if err != nil {
		t.Fatalf("Create alpha: %v", err)
	}
	_, err = sr.Create(ctx, "beta", nil)
	if err != nil {
		t.Fatalf("Create beta: %v", err)
	}

	records, err := sr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("List returned %d records, want 2", len(records))
	}

	names := map[string]bool{}
	for _, r := range records {
		names[r.ID] = true
	}
	if !names["alpha"] || !names["beta"] {
		t.Errorf("List names = %v, want alpha and beta", names)
	}
}

// --- Get ---

func TestSlotGet(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	_, err := sr.Create(ctx, "target", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rec, err := sr.Get(ctx, "target")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec.ID != "target" {
		t.Errorf("Get ID = %q, want %q", rec.ID, "target")
	}

	// Get nonexistent
	_, err = sr.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("Get(nonexistent): expected error, got nil")
	}
}

// --- Delete ---

func TestSlotDelete(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	created, err := sr.Create(ctx, "doomed", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	wtPath := created.Fields["path"].(string)

	err = sr.Delete(ctx, "doomed")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify worktree directory is gone
	if _, err := os.Stat(wtPath); err == nil {
		t.Error("worktree path still exists after Delete")
	}

	// Verify it's not in list anymore
	records, err := sr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	for _, r := range records {
		if r.ID == "doomed" {
			t.Error("deleted slot still appears in List")
		}
	}
}
