package projectresource

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	agentops "github.com/gh-xj/agentops"
	"github.com/gh-xj/agentops/dal"
	"github.com/gh-xj/agentops/resource"
)

// --- test helpers ---

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

func (e *realExec) Run(name string, args ...string) (string, error)           { return "", nil }
func (e *realExec) RunInDir(dir, name string, args ...string) (string, error) { return "", nil }
func (e *realExec) RunOsascript(script string) string                         { return "" }
func (e *realExec) Which(cmd string) bool                                     { return false }

func newTestResource(t *testing.T) (*ProjectResource, *agentops.AppContext) {
	t.Helper()
	pr := New(&realFS{}, &realExec{})
	ctx := agentops.NewAppContext(context.Background())
	return pr, ctx
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ resource.Resource = (*ProjectResource)(nil)
}

// --- Schema ---

func TestProjectSchema(t *testing.T) {
	pr := New(&realFS{}, &realExec{})
	s := pr.Schema()

	if s.Kind != "project" {
		t.Errorf("Schema().Kind = %q, want %q", s.Kind, "project")
	}
	if s.Description == "" {
		t.Error("Schema().Description is empty")
	}
	if len(s.CreateArgs) == 0 {
		t.Error("Schema().CreateArgs is empty")
	}
}

// --- Create ---

func TestProjectCreate(t *testing.T) {
	pr, ctx := newTestResource(t)
	tmp := t.TempDir()

	slug := "myapp"
	opts := map[string]string{
		"base_dir": tmp,
		"module":   "github.com/example/myapp",
	}

	rec, err := pr.Create(ctx, slug, opts)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if rec.Kind != "project" {
		t.Errorf("record Kind = %q, want %q", rec.Kind, "project")
	}
	if rec.ID != slug {
		t.Errorf("record ID = %q, want %q", rec.ID, slug)
	}

	createdPath, ok := rec.Fields["path"].(string)
	if !ok || createdPath == "" {
		t.Fatal("record missing path field")
	}

	// Verify expected files exist
	expectedFiles := []string{
		"main.go",
		"go.mod",
		"README.md",
		"cmd/root.go",
	}
	for _, f := range expectedFiles {
		fullPath := filepath.Join(createdPath, f)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("expected file %q not found", f)
		}
	}

	// Verify go.mod contains the module path
	goModData, err := os.ReadFile(filepath.Join(createdPath, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	if got := string(goModData); !contains(got, "github.com/example/myapp") {
		t.Errorf("go.mod does not contain module path, got:\n%s", got)
	}

	// Verify main.go imports the module
	mainData, err := os.ReadFile(filepath.Join(createdPath, "main.go"))
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if got := string(mainData); !contains(got, "github.com/example/myapp/cmd") {
		t.Errorf("main.go does not import cmd package, got:\n%s", got)
	}
}

func TestProjectCreateDefaultModule(t *testing.T) {
	pr, ctx := newTestResource(t)
	tmp := t.TempDir()

	slug := "coolproject"
	opts := map[string]string{
		"base_dir": tmp,
	}

	rec, err := pr.Create(ctx, slug, opts)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	createdPath := rec.Fields["path"].(string)
	goModData, err := os.ReadFile(filepath.Join(createdPath, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	// When no module is specified, slug is used as the module name
	if got := string(goModData); !contains(got, "module coolproject") {
		t.Errorf("go.mod does not contain default module path, got:\n%s", got)
	}
}

func TestProjectCreateEmptySlug(t *testing.T) {
	pr, ctx := newTestResource(t)
	tmp := t.TempDir()

	opts := map[string]string{
		"base_dir": tmp,
	}

	_, err := pr.Create(ctx, "", opts)
	if err == nil {
		t.Error("Create with empty slug: expected error, got nil")
	}
}

func TestProjectCreateDirExists(t *testing.T) {
	pr, ctx := newTestResource(t)
	tmp := t.TempDir()

	// Pre-create the target directory with a file in it
	projDir := filepath.Join(tmp, "existing")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "blocker.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := map[string]string{
		"base_dir": tmp,
	}

	_, err := pr.Create(ctx, "existing", opts)
	if err == nil {
		t.Error("Create into non-empty dir: expected error, got nil")
	}
}

// --- List ---

func TestProjectListEmpty(t *testing.T) {
	pr, ctx := newTestResource(t)

	records, err := pr.List(ctx, nil)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("List returned %d records, want 0", len(records))
	}
}

// --- Get ---

func TestProjectGetNotSupported(t *testing.T) {
	pr, ctx := newTestResource(t)

	_, err := pr.Get(ctx, "anything")
	if err == nil {
		t.Error("Get: expected error, got nil")
	}
}

// --- helpers ---

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
