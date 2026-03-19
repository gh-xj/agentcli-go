package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

func newScaffoldSvc() *ScaffoldService {
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	tpl := operator.NewTemplateOperator(fs)
	comp := operator.NewComplianceOperator(fs)
	return NewScaffoldService(tpl, comp, fs, exec)
}

func TestScaffoldService_New(t *testing.T) {
	svc := newScaffoldSvc()
	dir := t.TempDir()

	root, err := svc.New(dir, "myapp", "example.com/myapp", ScaffoldNewOptions{})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	expectedFiles := []string{
		"main.go",
		"cmd/root.go",
		"Taskfile.yml",
		"internal/io/output.go",
	}

	for _, f := range expectedFiles {
		abs := filepath.Join(root, f)
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestScaffoldService_NewLeanDefault(t *testing.T) {
	svc := newScaffoldSvc()
	dir := t.TempDir()

	root, err := svc.New(dir, "leanapp", "example.com/leanapp", ScaffoldNewOptions{})
	if err != nil {
		t.Fatalf("New lean failed: %v", err)
	}

	// Core files should exist
	for _, f := range []string{"main.go", "cmd/root.go", "Taskfile.yml", "internal/io/output.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); os.IsNotExist(err) {
			t.Errorf("expected %s to exist in lean mode", f)
		}
	}

	// DAG files should NOT exist
	for _, f := range []string{"service/container.go", "dal/interfaces.go", "operator/interfaces.go", "internal/app/bootstrap.go", "internal/config/schema.go", "pkg/version/version.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); !os.IsNotExist(err) {
			t.Errorf("expected %s to NOT exist in lean default", f)
		}
	}
}

func TestScaffoldService_NewFull(t *testing.T) {
	svc := newScaffoldSvc()
	dir := t.TempDir()

	root, err := svc.New(dir, "fullapp", "example.com/fullapp", ScaffoldNewOptions{Full: true})
	if err != nil {
		t.Fatalf("New full failed: %v", err)
	}

	// All files should exist including DAG scaffold
	for _, f := range []string{"main.go", "cmd/root.go", "service/container.go", "dal/interfaces.go", "operator/interfaces.go", "internal/app/bootstrap.go", "internal/config/schema.go", "pkg/version/version.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); os.IsNotExist(err) {
			t.Errorf("expected %s to exist in full mode", f)
		}
	}
}

func TestScaffoldService_NewMinimal(t *testing.T) {
	svc := newScaffoldSvc()
	dir := t.TempDir()

	root, err := svc.New(dir, "miniapp", "example.com/miniapp", ScaffoldNewOptions{Minimal: true})
	if err != nil {
		t.Fatalf("New minimal failed: %v", err)
	}

	// main.go should exist
	if _, err := os.Stat(filepath.Join(root, "main.go")); os.IsNotExist(err) {
		t.Error("expected main.go to exist in minimal mode")
	}

	// service/ should NOT exist
	if _, err := os.Stat(filepath.Join(root, "service")); !os.IsNotExist(err) {
		t.Error("expected service/ to NOT exist in minimal mode")
	}

	// dal/ should NOT exist
	if _, err := os.Stat(filepath.Join(root, "dal")); !os.IsNotExist(err) {
		t.Error("expected dal/ to NOT exist in minimal mode")
	}

	// operator/ should NOT exist
	if _, err := os.Stat(filepath.Join(root, "operator")); !os.IsNotExist(err) {
		t.Error("expected operator/ to NOT exist in minimal mode")
	}
}

func TestScaffoldService_NewEmptyName(t *testing.T) {
	svc := newScaffoldSvc()
	dir := t.TempDir()

	_, err := svc.New(dir, "", "example.com/x", ScaffoldNewOptions{})
	if err == nil {
		t.Error("expected error for empty name")
	}
}
