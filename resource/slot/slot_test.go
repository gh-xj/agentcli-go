package slotresource

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	agentops "github.com/gh-xj/agentops"
	"github.com/gh-xj/agentops/dal"
	"github.com/gh-xj/agentops/resource"
)

// setupGitRepo creates a temporary git repo with an initial commit.
// Returns the repo path.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main"},
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
func newTestResource(t *testing.T, projectDir string) (*SlotResource, *agentops.AppContext) {
	t.Helper()
	sr := New(&realFS{}, &realExec{})
	ctx := agentops.NewAppContext(context.Background())
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
	var _ resource.Doctor = (*SlotResource)(nil)
	var _ resource.Pruner = (*SlotResource)(nil)
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

// --- Config loading ---

func TestConfigLoadDefaults(t *testing.T) {
	// No config file exists — all defaults
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "myrepo")
	os.MkdirAll(filepath.Join(repoRoot, ".agentops"), 0o755)

	cfg, err := LoadSlotConfig(&realFS{}, filepath.Join(repoRoot, ".agentops"), repoRoot)
	if err != nil {
		t.Fatalf("LoadSlotConfig: %v", err)
	}
	if cfg.BaseBranch != "main" {
		t.Errorf("BaseBranch = %q, want %q", cfg.BaseBranch, "main")
	}
	if cfg.BranchPrefix != "slot" {
		t.Errorf("BranchPrefix = %q, want %q", cfg.BranchPrefix, "slot")
	}
	if cfg.MarkerFile != ".slot" {
		t.Errorf("MarkerFile = %q, want %q", cfg.MarkerFile, ".slot")
	}
	if cfg.WorktreePrefix != "myrepo" {
		t.Errorf("WorktreePrefix = %q, want %q", cfg.WorktreePrefix, "myrepo")
	}
	if len(cfg.Slots) != 0 {
		t.Errorf("Slots = %v, want empty", cfg.Slots)
	}
}

func TestConfigLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "myrepo")
	agentopsDir := filepath.Join(repoRoot, ".agentops")
	os.MkdirAll(agentopsDir, 0o755)

	yaml := `slots: [alpha, beta, gamma]
base_branch: develop
branch_prefix: wt
marker_file: .marker
worktree_prefix: custom
`
	os.WriteFile(filepath.Join(agentopsDir, "slot.yaml"), []byte(yaml), 0o644)

	cfg, err := LoadSlotConfig(&realFS{}, agentopsDir, repoRoot)
	if err != nil {
		t.Fatalf("LoadSlotConfig: %v", err)
	}
	if len(cfg.Slots) != 3 {
		t.Errorf("Slots len = %d, want 3", len(cfg.Slots))
	}
	if cfg.BaseBranch != "develop" {
		t.Errorf("BaseBranch = %q, want %q", cfg.BaseBranch, "develop")
	}
	if cfg.BranchPrefix != "wt" {
		t.Errorf("BranchPrefix = %q, want %q", cfg.BranchPrefix, "wt")
	}
	if cfg.MarkerFile != ".marker" {
		t.Errorf("MarkerFile = %q, want %q", cfg.MarkerFile, ".marker")
	}
	if cfg.WorktreePrefix != "custom" {
		t.Errorf("WorktreePrefix = %q, want %q", cfg.WorktreePrefix, "custom")
	}
}

func TestConfigValidateSlotName(t *testing.T) {
	cfg := &SlotConfig{
		Slots:        []string{"alpha", "beta"},
		BranchPrefix: "slot",
		MarkerFile:   ".slot",
		BaseBranch:   "main",
	}

	if err := cfg.ValidateSlotName("alpha"); err != nil {
		t.Errorf("unexpected error for valid slot: %v", err)
	}
	if err := cfg.ValidateSlotName("gamma"); err == nil {
		t.Error("expected error for invalid slot name")
	}
	if err := cfg.ValidateSlotName(""); err == nil {
		t.Error("expected error for empty slot name")
	}
}

func TestConfigSlotPaths(t *testing.T) {
	cfg := &SlotConfig{
		WorktreePrefix: "myrepo",
		BranchPrefix:   "slot",
		MarkerFile:     ".slot",
		BaseBranch:     "main",
	}

	repoRoot := "/home/user/repos/myrepo"
	paths := cfg.SlotPaths(repoRoot, "alpha")

	expectedWT := "/home/user/repos/worktrees/myrepo-alpha"
	if paths.WorktreePath != expectedWT {
		t.Errorf("WorktreePath = %q, want %q", paths.WorktreePath, expectedWT)
	}
	if paths.Branch != "slot/alpha" {
		t.Errorf("Branch = %q, want %q", paths.Branch, "slot/alpha")
	}
	expectedMarker := "/home/user/repos/worktrees/myrepo-alpha/.slot"
	if paths.MarkerPath != expectedMarker {
		t.Errorf("MarkerPath = %q, want %q", paths.MarkerPath, expectedMarker)
	}
	if paths.Name != "alpha" {
		t.Errorf("Name = %q, want %q", paths.Name, "alpha")
	}
}

func TestConfigMarkerFiles(t *testing.T) {
	cfg := &SlotConfig{MarkerFile: ".slot"}
	mf := cfg.MarkerFiles()
	if !mf[".slot"] {
		t.Error("expected .slot in marker files")
	}
	if mf[".other"] {
		t.Error("unexpected .other in marker files")
	}
}

// --- Git operations ---

func TestWorktreeListEmpty(t *testing.T) {
	repoDir := setupGitRepo(t)
	entries, err := WorktreeList(&realExec{}, repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (main worktree), got %d", len(entries))
	}
}

func TestParseWorktreeList(t *testing.T) {
	raw := `worktree /home/user/repo
HEAD abc123
branch refs/heads/main

worktree /home/user/worktrees/slot-alpha
HEAD def456
branch refs/heads/slot/alpha

`
	entries := parseWorktreeList(raw)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "/home/user/repo" {
		t.Errorf("entry[0].Path = %q", entries[0].Path)
	}
	if entries[0].Branch != "main" {
		t.Errorf("entry[0].Branch = %q, want %q", entries[0].Branch, "main")
	}
	if entries[1].Branch != "slot/alpha" {
		t.Errorf("entry[1].Branch = %q, want %q", entries[1].Branch, "slot/alpha")
	}
	if entries[0].Head != "abc123" {
		t.Errorf("entry[0].Head = %q, want %q", entries[0].Head, "abc123")
	}
}

func TestGitError(t *testing.T) {
	ge := &GitError{
		Args:   []string{"status", "--porcelain"},
		Output: "fatal: not a repo",
		Err:    os.ErrNotExist,
	}

	msg := ge.Error()
	if msg != "git status --porcelain: fatal: not a repo" {
		t.Errorf("unexpected error message: %s", msg)
	}
	if ge.Unwrap() != os.ErrNotExist {
		t.Error("Unwrap should return underlying error")
	}
}

func TestIsDirtyExcluding(t *testing.T) {
	t.Run("clean repo", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		dirty, err := IsDirtyExcluding(&realExec{}, repoDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if dirty {
			t.Fatal("fresh repo should not be dirty")
		}
	})

	t.Run("dirty repo", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("x"), 0o644)
		dirty, err := IsDirtyExcluding(&realExec{}, repoDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !dirty {
			t.Fatal("repo with untracked file should be dirty")
		}
	})

	t.Run("excluded file only", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		os.WriteFile(filepath.Join(repoDir, ".slot"), []byte("alpha"), 0o644)
		exclude := map[string]bool{".slot": true}
		dirty, err := IsDirtyExcluding(&realExec{}, repoDir, exclude)
		if err != nil {
			t.Fatal(err)
		}
		if dirty {
			t.Fatal("repo with only excluded file should not be dirty")
		}
	})

	t.Run("excluded plus dirty", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		os.WriteFile(filepath.Join(repoDir, ".slot"), []byte("alpha"), 0o644)
		os.WriteFile(filepath.Join(repoDir, "real-change.txt"), []byte("y"), 0o644)
		exclude := map[string]bool{".slot": true}
		dirty, err := IsDirtyExcluding(&realExec{}, repoDir, exclude)
		if err != nil {
			t.Fatal(err)
		}
		if !dirty {
			t.Fatal("repo with non-excluded changes should be dirty")
		}
	})
}

func TestCommitsBehind(t *testing.T) {
	repoDir := setupGitRepo(t)
	ex := &realExec{}

	// Create a worktree with a slot branch
	wtPath := filepath.Join(filepath.Dir(repoDir), "wt-behind")
	if err := WorktreeAdd(ex, repoDir, wtPath, "slot/behind-test"); err != nil {
		t.Fatal(err)
	}
	defer WorktreeRemove(ex, repoDir, wtPath)

	mainBranch := currentBranch(t, repoDir)

	// Initially 0 behind
	n, err := CommitsBehind(ex, repoDir, "slot/behind-test", mainBranch)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0 behind, got %d", n)
	}

	// Add commits on the main branch
	for i := 0; i < 3; i++ {
		c := exec.Command("git", "commit", "--allow-empty", "-m", "advance main")
		c.Dir = repoDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("commit: %s %v", out, err)
		}
	}

	n, err = CommitsBehind(ex, repoDir, "slot/behind-test", mainBranch)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("expected 3 behind, got %d", n)
	}
}

func TestListPrefixBranches(t *testing.T) {
	repoDir := setupGitRepo(t)
	ex := &realExec{}

	branches, err := ListPrefixBranches(ex, repoDir, "slot")
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected 0 slot branches, got %d", len(branches))
	}

	// Create two slot worktrees
	wt1 := filepath.Join(filepath.Dir(repoDir), "wt-slot1")
	wt2 := filepath.Join(filepath.Dir(repoDir), "wt-slot2")
	if err := WorktreeAdd(ex, repoDir, wt1, "slot/alpha"); err != nil {
		t.Fatal(err)
	}
	defer WorktreeRemove(ex, repoDir, wt1)
	if err := WorktreeAdd(ex, repoDir, wt2, "slot/beta"); err != nil {
		t.Fatal(err)
	}
	defer WorktreeRemove(ex, repoDir, wt2)

	branches, err = ListPrefixBranches(ex, repoDir, "slot")
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 slot branches, got %d: %v", len(branches), branches)
	}
}

// --- Doctor ---

func TestDoctorCleanSlot(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	// Create a clean slot
	_, err := sr.Create(ctx, "clean", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	checks, err := sr.Doctor(ctx)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	// Should have exactly one "ok" check for the clean slot
	found := false
	for _, c := range checks {
		if c.Name == "clean" && c.Status == "ok" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected ok check for clean slot, got: %+v", checks)
	}
}

func TestDoctorDirtySlot(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	rec, err := sr.Create(ctx, "dirty", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Make the slot dirty
	wtPath := rec.Fields["path"].(string)
	os.WriteFile(filepath.Join(wtPath, "uncommitted.txt"), []byte("dirty"), 0o644)

	checks, err := sr.Doctor(ctx)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "dirty" && c.Status == "dirty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected dirty check, got: %+v", checks)
	}
}

func TestDoctorMissingMarker(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	rec, err := sr.Create(ctx, "nomarker", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Remove the marker file
	wtPath := rec.Fields["path"].(string)
	os.Remove(filepath.Join(wtPath, ".slot"))

	checks, err := sr.Doctor(ctx)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "nomarker" && c.Status == "missing_marker" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_marker check, got: %+v", checks)
	}
}

func TestDoctorBehindBase(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	_, err := sr.Create(ctx, "behind", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Advance main branch
	mainBranch := currentBranch(t, repoDir)
	_ = mainBranch
	for i := 0; i < 2; i++ {
		c := exec.Command("git", "commit", "--allow-empty", "-m", "advance main")
		c.Dir = repoDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("commit: %s %v", out, err)
		}
	}

	checks, err := sr.Doctor(ctx)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Name == "behind" && c.Status == "behind" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected behind check, got: %+v", checks)
	}
}

func TestDoctorStaleBranch(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)
	ex := &realExec{}

	// Create a slot, then remove the worktree but leave the branch
	rec, err := sr.Create(ctx, "stale", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	wtPath := rec.Fields["path"].(string)

	// Force-remove the worktree directory without cleaning up the branch
	WorktreeRemove(ex, repoDir, wtPath)
	// The branch slot/stale still exists but has no worktree

	checks, err := sr.Doctor(ctx)
	if err != nil {
		t.Fatalf("Doctor: %v", err)
	}

	found := false
	for _, c := range checks {
		if c.Status == "stale_branch" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected stale_branch check, got: %+v", checks)
	}
}

// --- Prune ---

func TestPruneDryRun(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	_, err := sr.Create(ctx, "pruneme", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Dry-run prune (confirm=false)
	results, err := sr.Prune(ctx, false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Name == "pruneme" && r.Action == "would_remove" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected would_remove for pruneme, got: %+v", results)
	}

	// Verify worktree still exists (dry-run)
	records, err := sr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 slot after dry-run prune, got %d", len(records))
	}
}

func TestPruneConfirm(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	_, err := sr.Create(ctx, "pruneme", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Actual prune (confirm=true)
	results, err := sr.Prune(ctx, true)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Name == "pruneme" && r.Action == "removed" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected removed for pruneme, got: %+v", results)
	}

	// Verify worktree is gone
	records, err := sr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 slots after prune, got %d", len(records))
	}
}

func TestPruneSkipsDirty(t *testing.T) {
	repoDir := setupGitRepo(t)
	sr, ctx := newTestResource(t, repoDir)

	rec, err := sr.Create(ctx, "dirtypruneme", nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Make it dirty
	wtPath := rec.Fields["path"].(string)
	os.WriteFile(filepath.Join(wtPath, "uncommitted.txt"), []byte("dirty"), 0o644)

	results, err := sr.Prune(ctx, true)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Name == "dirtypruneme" && r.Action == "skipped" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected skipped for dirty slot, got: %+v", results)
	}

	// Verify worktree still exists
	records, err := sr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 slot after prune (skipped dirty), got %d", len(records))
	}
}

// --- ReadMarker ---

func TestReadMarker(t *testing.T) {
	dir := t.TempDir()
	fs := &realFS{}

	// Missing marker
	got := ReadMarker(fs, dir, ".slot")
	if got != "" {
		t.Errorf("ReadMarker (missing) = %q, want empty", got)
	}

	// Write marker and read it back
	os.WriteFile(filepath.Join(dir, ".slot"), []byte("alpha\n"), 0o644)
	got = ReadMarker(fs, dir, ".slot")
	if got != "alpha" {
		t.Errorf("ReadMarker = %q, want %q", got, "alpha")
	}
}

// currentBranch returns the current branch name in the repo.
func currentBranch(t *testing.T, repoDir string) string {
	t.Helper()
	c := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	c.Dir = repoDir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("get current branch: %s %v", out, err)
	}
	branch := string(out)
	for len(branch) > 0 && (branch[len(branch)-1] == '\n' || branch[len(branch)-1] == '\r') {
		branch = branch[:len(branch)-1]
	}
	return branch
}
