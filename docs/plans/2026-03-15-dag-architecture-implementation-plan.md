# DAG Architecture Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure agentcli-go into a DAG-layered architecture (cmd → service → operator → dal) with Wire DI, and update the scaffold to generate DAG-structured projects by default.

**Architecture:** Root package becomes the model layer (AppContext, Hook, CLIError stay). Implementation functions move to dal/ (I/O), operator/ (business logic), service/ (orchestration + Wire container). cmd/agentcli becomes the handler layer calling service.

**Tech Stack:** Go, google/wire, zerolog, cobra (via cobrax)

---

### Task 1: Add Wire dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add wire to go.mod**

Run: `cd /Users/xiangjun/Documents/codebase/xj-m/xj-scripts/agentcli-go && go get github.com/google/wire`

**Step 2: Verify**

Run: `grep wire go.mod`
Expected: `github.com/google/wire` appears in require block

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add google/wire dependency for DI"
```

---

### Task 2: Create dal/ layer — interfaces

**Files:**
- Create: `dal/interfaces.go`
- Test: `dal/interfaces_test.go`

**Step 1: Write the interface definitions**

```go
// dal/interfaces.go
package dal

import "io"

// FileSystem abstracts file and directory operations.
type FileSystem interface {
	Exists(path string) bool
	EnsureDir(dir string) error
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm int) error
	ReadDir(path string) ([]DirEntry, error)
	BaseName(path string) string
}

// DirEntry is a minimal directory entry.
type DirEntry struct {
	Name  string
	IsDir bool
}

// Executor abstracts command execution and PATH lookups.
type Executor interface {
	Run(name string, args ...string) (string, error)
	RunInDir(dir, name string, args ...string) (string, error)
	RunOsascript(script string) string
	Which(cmd string) bool
}

// Logger abstracts structured logger initialization.
type Logger interface {
	Init(verbose bool, w io.Writer)
}
```

**Step 2: Write a compile-check test**

```go
// dal/interfaces_test.go
package dal_test

import (
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
)

func TestInterfacesCompile(t *testing.T) {
	// Compile-time check that interfaces are importable
	var _ dal.FileSystem
	var _ dal.Executor
	var _ dal.Logger
}
```

**Step 3: Run test**

Run: `go test ./dal/...`
Expected: PASS

**Step 4: Commit**

```bash
git add dal/
git commit -m "feat(dal): add FileSystem, Executor, Logger interfaces"
```

---

### Task 3: Create dal/ layer — FileSystem implementation

**Files:**
- Create: `dal/filesystem.go`
- Test: `dal/filesystem_test.go`

**Step 1: Write failing test**

```go
// dal/filesystem_test.go
package dal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
)

func TestFileSystemImpl_Exists(t *testing.T) {
	fs := dal.NewFileSystem()
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.txt")
	if fs.Exists(f) {
		t.Fatal("expected file to not exist")
	}
	os.WriteFile(f, []byte("hi"), 0644)
	if !fs.Exists(f) {
		t.Fatal("expected file to exist")
	}
}

func TestFileSystemImpl_EnsureDir(t *testing.T) {
	fs := dal.NewFileSystem()
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "a", "b", "c")
	if err := fs.EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatal("expected directory to exist")
	}
}

func TestFileSystemImpl_ReadWriteFile(t *testing.T) {
	fs := dal.NewFileSystem()
	tmp := t.TempDir()
	f := filepath.Join(tmp, "data.txt")
	if err := fs.WriteFile(f, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := fs.ReadFile(f)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestFileSystemImpl_ReadDir(t *testing.T) {
	fs := dal.NewFileSystem()
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "a.txt"), []byte("a"), 0644)
	os.Mkdir(filepath.Join(tmp, "subdir"), 0755)
	entries, err := fs.ReadDir(tmp)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestFileSystemImpl_BaseName(t *testing.T) {
	fs := dal.NewFileSystem()
	if got := fs.BaseName("/foo/bar.txt"); got != "bar" {
		t.Fatalf("got %q, want %q", got, "bar")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./dal/...`
Expected: FAIL — `NewFileSystem` not defined

**Step 3: Write implementation**

```go
// dal/filesystem.go
package dal

import (
	"os"
	"path/filepath"
	"strings"
)

// FileSystemImpl is the real filesystem implementation.
type FileSystemImpl struct{}

// NewFileSystem returns a new FileSystemImpl.
func NewFileSystem() *FileSystemImpl {
	return &FileSystemImpl{}
}

func (f *FileSystemImpl) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *FileSystemImpl) EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func (f *FileSystemImpl) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *FileSystemImpl) WriteFile(path string, data []byte, perm int) error {
	return os.WriteFile(path, data, os.FileMode(perm))
}

func (f *FileSystemImpl) ReadDir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = DirEntry{Name: e.Name(), IsDir: e.IsDir()}
	}
	return result, nil
}

func (f *FileSystemImpl) BaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./dal/...`
Expected: PASS

**Step 5: Commit**

```bash
git add dal/
git commit -m "feat(dal): add FileSystem implementation"
```

---

### Task 4: Create dal/ layer — Executor implementation

**Files:**
- Create: `dal/exec.go`
- Test: `dal/exec_test.go`

**Step 1: Write failing test**

```go
// dal/exec_test.go
package dal_test

import (
	"strings"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
)

func TestExecutorImpl_Run(t *testing.T) {
	ex := dal.NewExecutor()
	out, err := ex.Run("echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("got %q, want contains 'hello'", out)
	}
}

func TestExecutorImpl_RunInDir(t *testing.T) {
	ex := dal.NewExecutor()
	out, err := ex.RunInDir("/tmp", "pwd")
	if err != nil {
		t.Fatalf("RunInDir: %v", err)
	}
	// /tmp may resolve to /private/tmp on macOS
	if !strings.Contains(out, "tmp") {
		t.Fatalf("got %q, expected to contain 'tmp'", out)
	}
}

func TestExecutorImpl_Which(t *testing.T) {
	ex := dal.NewExecutor()
	if !ex.Which("echo") {
		t.Fatal("expected echo to be in PATH")
	}
	if ex.Which("nonexistent-binary-xyz-12345") {
		t.Fatal("expected nonexistent binary to not be found")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./dal/...`
Expected: FAIL — `NewExecutor` not defined

**Step 3: Write implementation**

```go
// dal/exec.go
package dal

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ExecutorImpl is the real command executor.
type ExecutorImpl struct{}

// NewExecutor returns a new ExecutorImpl.
func NewExecutor() *ExecutorImpl {
	return &ExecutorImpl{}
}

func (e *ExecutorImpl) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (e *ExecutorImpl) RunInDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (e *ExecutorImpl) RunOsascript(script string) string {
	cmd := exec.Command("osascript", "-e", script)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func (e *ExecutorImpl) Which(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./dal/...`
Expected: PASS

**Step 5: Commit**

```bash
git add dal/
git commit -m "feat(dal): add Executor implementation"
```

---

### Task 5: Create dal/ layer — Logger implementation

**Files:**
- Create: `dal/logger.go`
- Test: `dal/logger_test.go`

**Step 1: Write failing test**

```go
// dal/logger_test.go
package dal_test

import (
	"bytes"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
)

func TestLoggerImpl_Init(t *testing.T) {
	l := dal.NewLogger()
	var buf bytes.Buffer
	// Should not panic
	l.Init(false, &buf)
	l.Init(true, &buf)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./dal/...`
Expected: FAIL — `NewLogger` not defined

**Step 3: Write implementation**

```go
// dal/logger.go
package dal

import (
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggerImpl sets up zerolog.
type LoggerImpl struct{}

// NewLogger returns a new LoggerImpl.
func NewLogger() *LoggerImpl {
	return &LoggerImpl{}
}

func (l *LoggerImpl) Init(verbose bool, w io.Writer) {
	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
	}
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        w,
		NoColor:    false,
		TimeFormat: "15:04:05",
	}).With().Timestamp().Logger().Level(level)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./dal/...`
Expected: PASS

**Step 5: Commit**

```bash
git add dal/
git commit -m "feat(dal): add Logger implementation"
```

---

### Task 6: Create operator/ layer — interfaces

**Files:**
- Create: `operator/interfaces.go`

**Step 1: Write interface definitions**

```go
// operator/interfaces.go
package operator

import (
	"context"

	agentcli "github.com/gh-xj/agentcli-go"
)

// TemplateOperator handles template rendering and project file generation.
type TemplateOperator interface {
	RenderTemplate(path, body string, data TemplateData) error
	KebabToCamel(in string) string
	DetectLocalReplaceLine() string
	ParseModulePath(goMod string) string
	ResolveParentModule(targetRoot string) (modulePath, moduleRoot string, err error)
}

// TemplateData is the data passed to scaffold templates.
type TemplateData struct {
	Module           string
	Name             string
	Description      string
	Preset           string
	GokitReplaceLine string
}

// ComplianceOperator checks project structure compliance.
type ComplianceOperator interface {
	CheckFileExists(rootDir, relPath string) *agentcli.DoctorFinding
	CheckFileContains(rootDir, relPath, code, want, msg string) *agentcli.DoctorFinding
	ValidateCommandName(name string) error
}

// ArgsOperator handles CLI argument parsing.
type ArgsOperator interface {
	Parse(args []string) map[string]string
	Require(args map[string]string, key, usage string) (string, error)
	Get(args map[string]string, key, defaultVal string) string
	HasFlag(args map[string]string, key string) bool
}

// ScaffoldFiles defines the set of files to generate for a project.
type ScaffoldFiles struct {
	Files      map[string]string // relPath -> template body
	WriteGoMod bool
}

// CommandPreset defines a scaffold command preset.
type CommandPreset struct {
	Name        string
	Description string
}
```

**Step 2: Run compile check**

Run: `go build ./operator/...`
Expected: success

**Step 3: Commit**

```bash
git add operator/
git commit -m "feat(operator): add TemplateOperator, ComplianceOperator, ArgsOperator interfaces"
```

---

### Task 7: Create operator/ layer — ArgsOperator implementation

**Files:**
- Create: `operator/args_op.go`
- Test: `operator/args_op_test.go`

**Step 1: Write failing test**

```go
// operator/args_op_test.go
package operator_test

import (
	"testing"

	"github.com/gh-xj/agentcli-go/operator"
)

func TestArgsOperatorImpl_Parse(t *testing.T) {
	op := operator.NewArgsOperator()
	result := op.Parse([]string{"--name", "foo", "--verbose"})
	if result["name"] != "foo" {
		t.Fatalf("got %q, want %q", result["name"], "foo")
	}
	if result["verbose"] != "true" {
		t.Fatalf("got %q, want %q", result["verbose"], "true")
	}
}

func TestArgsOperatorImpl_Require(t *testing.T) {
	op := operator.NewArgsOperator()
	args := map[string]string{"name": "foo"}
	val, err := op.Require(args, "name", "usage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "foo" {
		t.Fatalf("got %q, want %q", val, "foo")
	}
	_, err = op.Require(args, "missing", "usage")
	if err == nil {
		t.Fatal("expected error for missing required arg")
	}
}

func TestArgsOperatorImpl_Get(t *testing.T) {
	op := operator.NewArgsOperator()
	args := map[string]string{"name": "foo"}
	if got := op.Get(args, "name", "bar"); got != "foo" {
		t.Fatalf("got %q, want %q", got, "foo")
	}
	if got := op.Get(args, "missing", "bar"); got != "bar" {
		t.Fatalf("got %q, want %q", got, "bar")
	}
}

func TestArgsOperatorImpl_HasFlag(t *testing.T) {
	op := operator.NewArgsOperator()
	args := map[string]string{"verbose": "true", "name": "foo"}
	if !op.HasFlag(args, "verbose") {
		t.Fatal("expected verbose to be set")
	}
	if op.HasFlag(args, "name") {
		t.Fatal("expected name to not be a flag")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./operator/...`
Expected: FAIL — `NewArgsOperator` not defined

**Step 3: Write implementation**

```go
// operator/args_op.go
package operator

import (
	"fmt"
	"strings"
)

// ArgsOperatorImpl implements ArgsOperator.
type ArgsOperatorImpl struct{}

// NewArgsOperator returns a new ArgsOperatorImpl.
func NewArgsOperator() *ArgsOperatorImpl {
	return &ArgsOperatorImpl{}
}

func (a *ArgsOperatorImpl) Parse(args []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				result[key] = args[i+1]
				i++
			} else {
				result[key] = "true"
			}
		}
	}
	return result
}

func (a *ArgsOperatorImpl) Require(args map[string]string, key, usage string) (string, error) {
	if val, ok := args[key]; ok && val != "" {
		return val, nil
	}
	return "", fmt.Errorf("required flag missing: --%s (%s)", key, usage)
}

func (a *ArgsOperatorImpl) Get(args map[string]string, key, defaultVal string) string {
	if val, ok := args[key]; ok && val != "" {
		return val
	}
	return defaultVal
}

func (a *ArgsOperatorImpl) HasFlag(args map[string]string, key string) bool {
	val, ok := args[key]
	return ok && val == "true"
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./operator/...`
Expected: PASS

**Step 5: Commit**

```bash
git add operator/
git commit -m "feat(operator): add ArgsOperator implementation"
```

---

### Task 8: Create operator/ layer — TemplateOperator implementation

**Files:**
- Create: `operator/template_op.go`
- Test: `operator/template_op_test.go`

**Step 1: Write failing test**

```go
// operator/template_op_test.go
package operator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

func TestTemplateOperatorImpl_RenderTemplate(t *testing.T) {
	op := operator.NewTemplateOperator(dal.NewFileSystem())
	tmp := t.TempDir()
	path := filepath.Join(tmp, "out.go")
	body := "package {{.Name}}\n"
	err := op.RenderTemplate(path, body, operator.TemplateData{Name: "foo"})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "package foo\n" {
		t.Fatalf("got %q", got)
	}
}

func TestTemplateOperatorImpl_KebabToCamel(t *testing.T) {
	op := operator.NewTemplateOperator(dal.NewFileSystem())
	tests := []struct{ in, want string }{
		{"foo-bar", "FooBar"},
		{"hello", "Hello"},
		{"a-b-c", "ABC"},
	}
	for _, tc := range tests {
		if got := op.KebabToCamel(tc.in); got != tc.want {
			t.Errorf("KebabToCamel(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestTemplateOperatorImpl_ParseModulePath(t *testing.T) {
	op := operator.NewTemplateOperator(dal.NewFileSystem())
	gomod := "module github.com/example/foo\n\ngo 1.21\n"
	if got := op.ParseModulePath(gomod); got != "github.com/example/foo" {
		t.Fatalf("got %q", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./operator/...`
Expected: FAIL — `NewTemplateOperator` not defined

**Step 3: Write implementation**

```go
// operator/template_op.go
package operator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gh-xj/agentcli-go/dal"
)

// TemplateOperatorImpl implements TemplateOperator.
type TemplateOperatorImpl struct {
	fs dal.FileSystem
}

// NewTemplateOperator returns a new TemplateOperatorImpl.
func NewTemplateOperator(fs dal.FileSystem) *TemplateOperatorImpl {
	return &TemplateOperatorImpl{fs: fs}
}

func (t *TemplateOperatorImpl) RenderTemplate(path, body string, data TemplateData) error {
	if err := t.fs.EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("ensure dir for template: %w", err)
	}
	tpl, err := template.New(filepath.Base(path)).Parse(body)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

func (t *TemplateOperatorImpl) KebabToCamel(in string) string {
	parts := strings.Split(in, "-")
	for i := range parts {
		if len(parts[i]) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func (t *TemplateOperatorImpl) DetectLocalReplaceLine() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 16; i++ {
		modFile := filepath.Join(dir, "go.mod")
		data, err := t.fs.ReadFile(modFile)
		if err == nil && strings.Contains(string(data), "module github.com/gh-xj/agentcli-go") {
			return fmt.Sprintf("replace github.com/gh-xj/agentcli-go => %s", dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func (t *TemplateOperatorImpl) ParseModulePath(goMod string) string {
	lines := strings.Split(goMod, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "module ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
		value = strings.Trim(value, `"`)
		return value
	}
	return ""
}

func (t *TemplateOperatorImpl) ResolveParentModule(targetRoot string) (modulePath, moduleRoot string, err error) {
	dir := filepath.Clean(targetRoot)
	for {
		modFile := filepath.Join(dir, "go.mod")
		raw, readErr := t.fs.ReadFile(modFile)
		if readErr == nil {
			module := t.ParseModulePath(string(raw))
			if module == "" {
				return "", "", fmt.Errorf("invalid go.mod in parent module root: %s", modFile)
			}
			return module, dir, nil
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return "", "", fmt.Errorf("read parent go.mod: %w", readErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("parent go.mod not found for --in-existing-module target: %s", targetRoot)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./operator/...`
Expected: PASS

**Step 5: Commit**

```bash
git add operator/
git commit -m "feat(operator): add TemplateOperator implementation"
```

---

### Task 9: Create operator/ layer — ComplianceOperator implementation

**Files:**
- Create: `operator/compliance_op.go`
- Test: `operator/compliance_op_test.go`

**Step 1: Write failing test**

```go
// operator/compliance_op_test.go
package operator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

func TestComplianceOperatorImpl_CheckFileExists(t *testing.T) {
	op := operator.NewComplianceOperator(dal.NewFileSystem())
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "exists.go"), []byte("ok"), 0644)

	if f := op.CheckFileExists(tmp, "exists.go"); f != nil {
		t.Fatalf("expected no finding for existing file, got %v", f)
	}
	if f := op.CheckFileExists(tmp, "missing.go"); f == nil {
		t.Fatal("expected finding for missing file")
	}
}

func TestComplianceOperatorImpl_CheckFileContains(t *testing.T) {
	op := operator.NewComplianceOperator(dal.NewFileSystem())
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "root.go"), []byte("import cobrax\n"), 0644)

	if f := op.CheckFileContains(tmp, "root.go", "code", "cobrax", "msg"); f != nil {
		t.Fatalf("expected no finding, got %v", f)
	}
	if f := op.CheckFileContains(tmp, "root.go", "code", "missing", "msg"); f == nil {
		t.Fatal("expected finding for missing content")
	}
}

func TestComplianceOperatorImpl_ValidateCommandName(t *testing.T) {
	op := operator.NewComplianceOperator(dal.NewFileSystem())
	if err := op.ValidateCommandName("foo-bar"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := op.ValidateCommandName("FooBar"); err == nil {
		t.Fatal("expected invalid for PascalCase")
	}
	if err := op.ValidateCommandName(""); err == nil {
		t.Fatal("expected invalid for empty")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./operator/...`
Expected: FAIL — `NewComplianceOperator` not defined

**Step 3: Write implementation**

```go
// operator/compliance_op.go
package operator

import (
	"fmt"
	"regexp"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
)

var validCommandName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ComplianceOperatorImpl implements ComplianceOperator.
type ComplianceOperatorImpl struct {
	fs dal.FileSystem
}

// NewComplianceOperator returns a new ComplianceOperatorImpl.
func NewComplianceOperator(fs dal.FileSystem) *ComplianceOperatorImpl {
	return &ComplianceOperatorImpl{fs: fs}
}

func (c *ComplianceOperatorImpl) CheckFileExists(rootDir, relPath string) *agentcli.DoctorFinding {
	fullPath := rootDir + "/" + relPath
	if c.fs.Exists(fullPath) {
		return nil
	}
	return &agentcli.DoctorFinding{
		Code:    "missing_file",
		Path:    relPath,
		Message: "required file is missing",
	}
}

func (c *ComplianceOperatorImpl) CheckFileContains(rootDir, relPath, code, want, msg string) *agentcli.DoctorFinding {
	fullPath := rootDir + "/" + relPath
	content, err := c.fs.ReadFile(fullPath)
	if err != nil {
		return nil // file missing is caught by CheckFileExists
	}
	if !strings.Contains(string(content), want) {
		return &agentcli.DoctorFinding{
			Code:    code,
			Path:    relPath,
			Message: msg,
		}
	}
	return nil
}

func (c *ComplianceOperatorImpl) ValidateCommandName(name string) error {
	if !validCommandName.MatchString(name) {
		return fmt.Errorf("invalid command name %q: use kebab-case [a-z0-9-]", name)
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./operator/...`
Expected: PASS

**Step 5: Commit**

```bash
git add operator/
git commit -m "feat(operator): add ComplianceOperator implementation"
```

---

### Task 10: Create service/ layer — container and Wire

**Files:**
- Create: `service/container.go`
- Create: `service/wire.go`

**Step 1: Write container**

```go
// service/container.go
package service

import (
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

// Container holds all wired dependencies.
type Container struct {
	FS   dal.FileSystem
	Exec dal.Executor
	Log  dal.Logger

	TemplateOp   operator.TemplateOperator
	ComplianceOp operator.ComplianceOperator
	ArgsOp       operator.ArgsOperator

	ScaffoldSvc  *ScaffoldService
	DoctorSvc    *DoctorService
	LifecycleSvc *LifecycleService
}

// NewContainer constructs the container from wired dependencies.
func NewContainer(
	fs dal.FileSystem,
	exec dal.Executor,
	lg dal.Logger,
	tpl operator.TemplateOperator,
	comp operator.ComplianceOperator,
	args operator.ArgsOperator,
	scaffold *ScaffoldService,
	doctor *DoctorService,
	lifecycle *LifecycleService,
) *Container {
	return &Container{
		FS:           fs,
		Exec:         exec,
		Log:          lg,
		TemplateOp:   tpl,
		ComplianceOp: comp,
		ArgsOp:       args,
		ScaffoldSvc:  scaffold,
		DoctorSvc:    doctor,
		LifecycleSvc: lifecycle,
	}
}

var globalContainer *Container

// Get returns the global container, initializing it on first call.
func Get() *Container {
	if globalContainer == nil {
		globalContainer = InitializeContainer()
	}
	return globalContainer
}

// Reset clears the global container (for testing).
func Reset() {
	globalContainer = nil
}
```

**Step 2: Write wire injector**

```go
//go:build wireinject

// service/wire.go
package service

import (
	"github.com/google/wire"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

var ProviderSet = wire.NewSet(
	// DAL
	dal.NewFileSystem,
	wire.Bind(new(dal.FileSystem), new(*dal.FileSystemImpl)),
	dal.NewExecutor,
	wire.Bind(new(dal.Executor), new(*dal.ExecutorImpl)),
	dal.NewLogger,
	wire.Bind(new(dal.Logger), new(*dal.LoggerImpl)),

	// Operators
	operator.NewTemplateOperator,
	wire.Bind(new(operator.TemplateOperator), new(*operator.TemplateOperatorImpl)),
	operator.NewComplianceOperator,
	wire.Bind(new(operator.ComplianceOperator), new(*operator.ComplianceOperatorImpl)),
	operator.NewArgsOperator,
	wire.Bind(new(operator.ArgsOperator), new(*operator.ArgsOperatorImpl)),

	// Services
	NewScaffoldService,
	NewDoctorService,
	NewLifecycleService,

	// Container
	NewContainer,
)

func InitializeContainer() *Container {
	wire.Build(ProviderSet)
	return nil
}
```

**Step 3: Verify it compiles (wire.go is build-tagged, so regular build skips it)**

Run: `go build ./service/...`
Expected: This will fail because ScaffoldService, DoctorService, LifecycleService don't exist yet. That's OK — we create them in the next tasks. For now, commit wire.go and container.go as scaffolds.

**Step 4: Commit**

```bash
git add service/
git commit -m "feat(service): add Wire container and injector scaffold"
```

---

### Task 11: Create service/ layer — LifecycleService

**Files:**
- Create: `service/lifecycle.go`
- Test: `service/lifecycle_test.go`

**Step 1: Write failing test**

```go
// service/lifecycle_test.go
package service_test

import (
	"context"
	"errors"
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/service"
)

type mockHook struct {
	preErr  error
	postErr error
	preCalled  bool
	postCalled bool
}

func (m *mockHook) Preflight(app *agentcli.AppContext) error {
	m.preCalled = true
	return m.preErr
}

func (m *mockHook) Postflight(app *agentcli.AppContext) error {
	m.postCalled = true
	return m.postErr
}

func TestLifecycleService_Run_HappyPath(t *testing.T) {
	svc := service.NewLifecycleService()
	hook := &mockHook{}
	ran := false
	err := svc.Run(agentcli.NewAppContext(context.Background()), hook, func(app *agentcli.AppContext) error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hook.preCalled || !hook.postCalled || !ran {
		t.Fatal("expected all phases to run")
	}
}

func TestLifecycleService_Run_PreflightError(t *testing.T) {
	svc := service.NewLifecycleService()
	hook := &mockHook{preErr: errors.New("preflight fail")}
	ran := false
	err := svc.Run(agentcli.NewAppContext(context.Background()), hook, func(app *agentcli.AppContext) error {
		ran = true
		return nil
	})
	if err == nil || !errors.Is(err, hook.preErr) {
		t.Fatalf("expected preflight error, got %v", err)
	}
	if ran {
		t.Fatal("run should not execute after preflight failure")
	}
}

func TestLifecycleService_Run_NilHook(t *testing.T) {
	svc := service.NewLifecycleService()
	ran := false
	err := svc.Run(agentcli.NewAppContext(context.Background()), nil, func(app *agentcli.AppContext) error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Fatal("expected run to execute")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./service/...`
Expected: FAIL — `NewLifecycleService` not defined

**Step 3: Write implementation**

```go
// service/lifecycle.go
package service

import (
	"context"

	agentcli "github.com/gh-xj/agentcli-go"
)

// LifecycleService orchestrates preflight/run/postflight execution.
type LifecycleService struct{}

// NewLifecycleService returns a new LifecycleService.
func NewLifecycleService() *LifecycleService {
	return &LifecycleService{}
}

// Run executes preflight, run, and postflight in order.
func (s *LifecycleService) Run(app *agentcli.AppContext, hook agentcli.Hook, run func(*agentcli.AppContext) error) error {
	if app == nil {
		app = agentcli.NewAppContext(context.TODO())
	}
	if hook != nil {
		if err := hook.Preflight(app); err != nil {
			return err
		}
	}
	if run != nil {
		if err := run(app); err != nil {
			return err
		}
	}
	if hook != nil {
		if err := hook.Postflight(app); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./service/...`
Expected: PASS

**Step 5: Commit**

```bash
git add service/
git commit -m "feat(service): add LifecycleService"
```

---

### Task 12: Create service/ layer — DoctorService

**Files:**
- Create: `service/doctor.go`
- Test: `service/doctor_test.go`

**Step 1: Write failing test**

```go
// service/doctor_test.go
package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
	"github.com/gh-xj/agentcli-go/service"
)

func setupDoctorProject(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	files := map[string]string{
		"main.go":                             "package main",
		"cmd/root.go":                         "import \"github.com/gh-xj/agentcli-go/cobrax\"\n// agentcli:add-command\n",
		"internal/app/bootstrap.go":           "package app",
		"internal/app/lifecycle.go":           "func Preflight() {}\nfunc Postflight() {}",
		"internal/app/errors.go":              "package app",
		"internal/config/schema.go":           "package config",
		"internal/config/load.go":             "package config",
		"internal/io/output.go":               "package appio",
		"internal/tools/smokecheck/main.go":   "package main",
		"pkg/version/version.go":              "package version",
		"test/e2e/cli_test.go":                "package e2e",
		"test/smoke/version.schema.json":      `{"schema_version": "v1", "required_keys": ["name"]}`,
		"Taskfile.yml":                         "tasks:\n  ci:\n  verify:\n  smoke:\n    cmds:\n      - test/smoke/version.output.json\n      - internal/tools/smokecheck\n",
		"go.mod":                               "module example\ngo 1.21",
		"service/wire.go":                      "package service\nvar ProviderSet",
		"dal/interfaces.go":                    "package dal",
		"operator/interfaces.go":               "package operator",
	}
	for relPath, content := range files {
		abs := filepath.Join(tmp, relPath)
		os.MkdirAll(filepath.Dir(abs), 0755)
		os.WriteFile(abs, []byte(content), 0644)
	}
	return tmp
}

func TestDoctorService_Run_AllPass(t *testing.T) {
	tmp := setupDoctorProject(t)
	fs := dal.NewFileSystem()
	comp := operator.NewComplianceOperator(fs)
	svc := service.NewDoctorService(comp, fs)
	report := svc.Run(tmp)
	if !report.OK {
		t.Fatalf("expected OK, got findings: %+v", report.Findings)
	}
}

func TestDoctorService_Run_MissingFile(t *testing.T) {
	tmp := setupDoctorProject(t)
	os.Remove(filepath.Join(tmp, "cmd/root.go"))
	fs := dal.NewFileSystem()
	comp := operator.NewComplianceOperator(fs)
	svc := service.NewDoctorService(comp, fs)
	report := svc.Run(tmp)
	if report.OK {
		t.Fatal("expected findings for missing cmd/root.go")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./service/...`
Expected: FAIL — `NewDoctorService` not defined

**Step 3: Write implementation**

```go
// service/doctor.go
package service

import (
	"slices"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

const rootCommandMarker = "// agentcli:add-command"

// DoctorService orchestrates project compliance checks.
type DoctorService struct {
	comp operator.ComplianceOperator
	fs   dal.FileSystem
}

// NewDoctorService returns a new DoctorService.
func NewDoctorService(comp operator.ComplianceOperator, fs dal.FileSystem) *DoctorService {
	return &DoctorService{comp: comp, fs: fs}
}

// Run checks whether a project follows the golden scaffold contract.
func (s *DoctorService) Run(rootDir string) agentcli.DoctorReport {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}

	report := agentcli.DoctorReport{SchemaVersion: "v1", OK: true, Findings: make([]agentcli.DoctorFinding, 0)}

	required := []string{
		"main.go",
		"cmd/root.go",
		"internal/app/bootstrap.go",
		"internal/app/lifecycle.go",
		"internal/app/errors.go",
		"internal/config/schema.go",
		"internal/config/load.go",
		"internal/io/output.go",
		"internal/tools/smokecheck/main.go",
		"pkg/version/version.go",
		"test/e2e/cli_test.go",
		"test/smoke/version.schema.json",
		"Taskfile.yml",
	}

	if s.fs.Exists(rootDir + "/go.mod") {
		required = append(required, "go.mod")
	}

	for _, p := range required {
		if f := s.comp.CheckFileExists(rootDir, p); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}

	contentChecks := []struct {
		path, code, want, msg string
	}{
		{"cmd/root.go", "missing_contract", `"github.com/gh-xj/agentcli-go/cobrax"`, "cobrax runtime import missing"},
		{"cmd/root.go", "missing_contract", rootCommandMarker, "missing scaffold command marker"},
		{"Taskfile.yml", "missing_contract", "ci:", "canonical CI task missing"},
		{"Taskfile.yml", "missing_contract", "verify:", "local verification task missing"},
		{"Taskfile.yml", "missing_contract", "test/smoke/version.output.json", "smoke artifact output path missing"},
		{"Taskfile.yml", "missing_contract", "internal/tools/smokecheck", "smoke schema validation command missing"},
		{"internal/app/lifecycle.go", "missing_contract", "Preflight", "lifecycle preflight hook missing"},
		{"internal/app/lifecycle.go", "missing_contract", "Postflight", "lifecycle postflight hook missing"},
		{"test/smoke/version.schema.json", "missing_contract", `"schema_version": "v1"`, "smoke schema version missing"},
		// DAG compliance checks
		{"service/wire.go", "missing_contract", "ProviderSet", "Wire provider set missing"},
		{"dal/interfaces.go", "missing_dag", "dal", "DAL interfaces file missing or empty"},
		{"operator/interfaces.go", "missing_dag", "operator", "operator interfaces file missing or empty"},
	}

	for _, c := range contentChecks {
		if f := s.comp.CheckFileContains(rootDir, c.path, c.code, c.want, c.msg); f != nil {
			report.Findings = append(report.Findings, *f)
		}
	}

	slices.SortFunc(report.Findings, func(a, b agentcli.DoctorFinding) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return strings.Compare(a.Code, b.Code)
	})
	report.OK = len(report.Findings) == 0
	return report
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./service/...`
Expected: PASS

**Step 5: Commit**

```bash
git add service/
git commit -m "feat(service): add DoctorService with DAG compliance checks"
```

---

### Task 13: Create service/ layer — ScaffoldService

**Files:**
- Create: `service/scaffold.go`
- Create: `service/templates.go` (move template constants here)
- Test: `service/scaffold_test.go`

**Step 1: Write failing test**

```go
// service/scaffold_test.go
package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
	"github.com/gh-xj/agentcli-go/service"
)

func TestScaffoldService_New(t *testing.T) {
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	tpl := operator.NewTemplateOperator(fs)
	comp := operator.NewComplianceOperator(fs)
	svc := service.NewScaffoldService(tpl, comp, fs, exec)

	tmp := t.TempDir()
	root, err := svc.New(tmp, "test-cli", "github.com/test/test-cli", service.ScaffoldNewOptions{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Check key files exist
	for _, f := range []string{"main.go", "cmd/root.go", "service/container.go", "dal/interfaces.go", "operator/interfaces.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("expected %s to exist", f)
		}
	}
}

func TestScaffoldService_New_Minimal(t *testing.T) {
	fs := dal.NewFileSystem()
	exec := dal.NewExecutor()
	tpl := operator.NewTemplateOperator(fs)
	comp := operator.NewComplianceOperator(fs)
	svc := service.NewScaffoldService(tpl, comp, fs, exec)

	tmp := t.TempDir()
	root, err := svc.New(tmp, "tiny", "github.com/test/tiny", service.ScaffoldNewOptions{Minimal: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "main.go")); err != nil {
		t.Error("expected main.go to exist")
	}
	// Minimal should NOT have DAG dirs
	if _, err := os.Stat(filepath.Join(root, "service")); err == nil {
		t.Error("minimal should not have service/")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./service/...`
Expected: FAIL — `NewScaffoldService` not defined

**Step 3: Write templates.go** (move all template constants from scaffold.go)

Create `service/templates.go` containing all the `const xxxTpl` values currently in `scaffold.go`. Also add new DAG-specific templates for the generated `service/container.go`, `service/wire.go`, `dal/interfaces.go`, `dal/filesystem.go`, `operator/interfaces.go`, `operator/example_op.go`.

The new DAG templates should follow the patterns established in the design doc. The existing templates (mainTpl, rootCmdTpl, etc.) move here verbatim.

**Step 4: Write scaffold.go**

```go
// service/scaffold.go
package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

// ScaffoldNewOptions controls scaffold generation behavior.
type ScaffoldNewOptions struct {
	InExistingModule bool
	Minimal          bool
}

// ScaffoldService orchestrates project scaffolding.
type ScaffoldService struct {
	tpl  operator.TemplateOperator
	comp operator.ComplianceOperator
	fs   dal.FileSystem
	exec dal.Executor
}

// NewScaffoldService returns a new ScaffoldService.
func NewScaffoldService(
	tpl operator.TemplateOperator,
	comp operator.ComplianceOperator,
	fs dal.FileSystem,
	exec dal.Executor,
) *ScaffoldService {
	return &ScaffoldService{tpl: tpl, comp: comp, fs: fs, exec: exec}
}

// New creates a new CLI project using the golden agentcli layout.
func (s *ScaffoldService) New(baseDir, name, module string, opts ScaffoldNewOptions) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("project name is required")
	}
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	module = strings.TrimSpace(module)
	writeGoMod := true

	root := filepath.Join(baseDir, name)
	if s.fs.Exists(root) {
		entries, err := s.fs.ReadDir(root)
		if err != nil {
			return "", err
		}
		if len(entries) > 0 {
			return "", fmt.Errorf("target directory is not empty: %s", root)
		}
	} else {
		if err := s.fs.EnsureDir(root); err != nil {
			return "", err
		}
	}

	if opts.InExistingModule {
		if module != "" {
			return "", errors.New("--module cannot be used with --in-existing-module")
		}
		modulePath, moduleRoot, err := s.tpl.ResolveParentModule(root)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(moduleRoot, root)
		if err != nil {
			return "", fmt.Errorf("resolve path under parent module: %w", err)
		}
		module = modulePath
		if rel != "." {
			module = module + "/" + filepath.ToSlash(rel)
		}
		writeGoMod = false
	} else if module == "" {
		module = name
	}

	files := map[string]string{
		"main.go":     mainTpl,
		"cmd/root.go": rootCmdTpl,
	}

	if opts.Minimal {
		files["README.md"] = minimalReadmeTpl
	} else {
		// Existing scaffold files
		files["internal/app/bootstrap.go"] = appBootstrapTpl
		files["internal/app/lifecycle.go"] = appLifecycleTpl
		files["internal/app/errors.go"] = appErrorsTpl
		files["internal/config/schema.go"] = configSchemaTpl
		files["internal/config/load.go"] = configLoadTpl
		files["internal/io/output.go"] = outputTpl
		files["internal/tools/smokecheck/main.go"] = smokeCheckTpl
		files["pkg/version/version.go"] = versionTpl
		files["test/e2e/cli_test.go"] = e2eTestTpl
		files["test/smoke/version.schema.json"] = smokeSchemaTpl
		files["Taskfile.yml"] = taskfileTpl
		files["README.md"] = readmeTpl
		// DAG layer files
		files["service/container.go"] = scaffoldContainerTpl
		files["service/wire.go"] = scaffoldWireTpl
		files["dal/interfaces.go"] = scaffoldDALInterfacesTpl
		files["dal/filesystem.go"] = scaffoldDALFilesystemTpl
		files["operator/interfaces.go"] = scaffoldOperatorInterfacesTpl
		files["operator/example_op.go"] = scaffoldOperatorExampleTpl
	}
	if writeGoMod {
		files["go.mod"] = goModTpl
	}

	cliName := filepath.Base(strings.TrimSpace(name))
	if cliName == "" || cliName == "." || cliName == string(filepath.Separator) {
		return "", errors.New("project name is required")
	}

	data := operator.TemplateData{
		Module:           module,
		Name:             cliName,
		GokitReplaceLine: s.tpl.DetectLocalReplaceLine(),
	}
	for path, body := range files {
		if err := s.tpl.RenderTemplate(filepath.Join(root, path), body, data); err != nil {
			return "", fmt.Errorf("render %s: %w", path, err)
		}
	}

	if writeGoMod {
		if _, err := s.exec.RunInDir(root, "go", "mod", "tidy"); err != nil {
			return "", fmt.Errorf("go mod tidy: %w", err)
		}
	}
	return root, nil
}

// AddCommand creates a command file and wires it into cmd/root.go.
func (s *ScaffoldService) AddCommand(rootDir, commandName, description, preset string) error {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	if err := s.comp.ValidateCommandName(commandName); err != nil {
		return err
	}
	description = strings.TrimSpace(description)
	preset = strings.TrimSpace(preset)
	if preset != "" {
		presetDesc, ok := commandPresets[preset]
		if !ok {
			return fmt.Errorf("invalid preset %q", preset)
		}
		if description == "" {
			description = presetDesc
		}
	}
	if description == "" {
		description = fmt.Sprintf("describe %s", commandName)
	}

	funcName := s.tpl.KebabToCamel(commandName)
	cmdFile := filepath.Join(rootDir, "cmd", commandName+".go")
	if s.fs.Exists(cmdFile) {
		return fmt.Errorf("command file already exists: %s", cmdFile)
	}
	if err := s.tpl.RenderTemplate(cmdFile, addCommandTpl, operator.TemplateData{
		Name:        commandName,
		Module:      funcName,
		Description: description,
		Preset:      preset,
	}); err != nil {
		return err
	}

	rootFile := filepath.Join(rootDir, "cmd", "root.go")
	content, err := s.fs.ReadFile(rootFile)
	if err != nil {
		return err
	}
	registerLine := fmt.Sprintf("registerCommand(%q, %sCommand())", commandName, funcName)
	text := string(content)
	if strings.Contains(text, registerLine) {
		return nil
	}
	idx := strings.Index(text, rootCommandMarker)
	if idx < 0 {
		return fmt.Errorf("marker %q not found in %s", rootCommandMarker, rootFile)
	}
	updated := text[:idx] + registerLine + "\n\t" + text[idx:]
	return s.fs.WriteFile(rootFile, []byte(updated), 0644)
}

var commandPresets = map[string]string{
	"file-sync":                "sync files between source and destination",
	"http-client":              "send HTTP requests to a target endpoint",
	"deploy-helper":            "run deterministic deploy workflow checks",
	"task-replay-orchestrator": "orchestrate external repo task runs with env injection and timeout hooks",
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./service/...`
Expected: PASS

**Step 6: Commit**

```bash
git add service/
git commit -m "feat(service): add ScaffoldService with DAG template generation"
```

---

### Task 14: Generate wire_gen.go

**Files:**
- Create: `service/wire_gen.go` (auto-generated)

**Step 1: Install wire if needed**

Run: `go install github.com/google/wire/cmd/wire@latest`

**Step 2: Generate wire_gen.go**

Run: `cd /Users/xiangjun/Documents/codebase/xj-m/xj-scripts/agentcli-go && wire ./service/`
Expected: writes `service/wire_gen.go`

**Step 3: Verify build**

Run: `go build ./service/...`
Expected: success

**Step 4: Commit**

```bash
git add service/wire_gen.go
git commit -m "chore(service): generate wire_gen.go"
```

---

### Task 15: Add backward-compatible shims in root package

The root package functions (`RunLifecycle`, `ScaffoldNew`, etc.) must continue to work but delegate to the service layer. This prevents breaking downstream consumers immediately.

**Files:**
- Modify: `lifecycle.go` — delegate to `service.Get().LifecycleSvc.Run()`
- Modify: `scaffold.go` — delegate `ScaffoldNew`, `ScaffoldAddCommand`, `Doctor` to service layer
- Modify: `args.go` — delegate to `operator.ArgsOperatorImpl` (or keep as-is since these are pure functions)
- Modify: `exec.go` — delegate to `dal.ExecutorImpl`
- Modify: `fs.go` — delegate to `dal.FileSystemImpl`
- Modify: `log.go` — delegate to `dal.LoggerImpl`

**Step 1: Update lifecycle.go**

Keep the same signature but delegate internally:
```go
func RunLifecycle(app *AppContext, hook Hook, run func(*AppContext) error) error {
	return service.Get().LifecycleSvc.Run(app, hook, run)
}
```

Note: This creates an import cycle (root imports service, service imports root). To avoid this, keep `RunLifecycle` as a standalone function in root that duplicates the logic (it's only 15 lines). The service layer's `LifecycleService` is for new code paths. Mark the root-level functions with a `// Deprecated:` comment directing users to the service layer.

**Step 2: Add deprecation comments to root-level implementation functions**

Add `// Deprecated: Use service.Get().ScaffoldSvc.New() instead.` to `ScaffoldNew`, `ScaffoldNewWithOptions`, `ScaffoldAddCommand`, `Doctor`.

Add `// Deprecated: Use dal.NewExecutor() instead.` to `RunCommand`, `RunOsascript`, `Which`, `CheckDependency`.

Add `// Deprecated: Use dal.NewFileSystem() instead.` to `FileExists`, `EnsureDir`, `GetBaseName`.

Add `// Deprecated: Use dal.NewLogger() instead.` to `InitLogger`.

Add `// Deprecated: Use operator.NewArgsOperator() instead.` to `ParseArgs`, `RequireArg`, `GetArg`, `HasFlag`.

Do NOT change any function signatures or behavior — only add comments.

**Step 3: Verify all existing tests pass**

Run: `go test ./...`
Expected: PASS — no behavioral changes

**Step 4: Commit**

```bash
git add *.go
git commit -m "chore: add deprecation notices for root-level functions, pointing to DAG layers"
```

---

### Task 16: Update cmd/agentcli to use service layer

**Files:**
- Modify: `cmd/agentcli/main.go`

**Step 1: Update the `new`, `add`, and `doctor` commands to use `service.Get()`**

Replace direct calls to `agentcli.ScaffoldNew(...)` with `service.Get().ScaffoldSvc.New(...)`.
Replace `agentcli.ScaffoldAddCommand(...)` with `service.Get().ScaffoldSvc.AddCommand(...)`.
Replace `agentcli.Doctor(...)` with `service.Get().DoctorSvc.Run(...)`.

The `ScaffoldNewOptions` type moves to the service package, so update the import.

**Step 2: Run existing tests**

Run: `go test ./cmd/agentcli/...`
Expected: PASS

**Step 3: Run full test suite**

Run: `go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/agentcli/main.go
git commit -m "refactor(cmd): use service layer for scaffold, doctor commands"
```

---

### Task 17: Write DAG scaffold templates

**Files:**
- Create or update: `service/templates.go`

Add these new template constants for the DAG files generated by `agentcli new`:

- `scaffoldContainerTpl` — generates `service/container.go` in the new project
- `scaffoldWireTpl` — generates `service/wire.go` in the new project
- `scaffoldDALInterfacesTpl` — generates `dal/interfaces.go` in the new project
- `scaffoldDALFilesystemTpl` — generates `dal/filesystem.go` in the new project
- `scaffoldOperatorInterfacesTpl` — generates `operator/interfaces.go` in the new project
- `scaffoldOperatorExampleTpl` — generates `operator/example_op.go` with a TODO stub

Also update `taskfileTpl` to include a `wire:` task.

**Step 1: Write the templates**

Each template should follow the patterns from xj_ops: interfaces in `interfaces.go`, implementations in separate files, Wire bindings in `wire.go`, container with `Get()` singleton.

**Step 2: Run scaffold test**

Run: `go test ./service/... -run TestScaffoldService`
Expected: PASS — generated projects include DAG files

**Step 3: Commit**

```bash
git add service/templates.go
git commit -m "feat(service): add DAG scaffold templates for generated projects"
```

---

### Task 18: Update existing tests for new scaffold output

**Files:**
- Modify: `cmd/agentcli/main_test.go`
- Modify: `scaffold_test.go`

**Step 1: Update scaffold tests to expect DAG directories**

Tests that check generated project structure need to verify:
- `service/container.go` exists
- `service/wire.go` exists
- `dal/interfaces.go` exists
- `operator/interfaces.go` exists
- Doctor report passes on freshly-generated projects

**Step 2: Update doctor tests to include DAG checks**

Tests need to verify:
- Missing `service/wire.go` produces a finding
- Missing `dal/interfaces.go` produces a finding
- Missing `operator/interfaces.go` produces a finding

**Step 3: Run full test suite**

Run: `go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/agentcli/main_test.go scaffold_test.go
git commit -m "test: update scaffold and doctor tests for DAG architecture"
```

---

### Task 19: Update CLAUDE.md and documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update architecture table**

Add new entries for `dal/`, `operator/`, `service/` packages and their purposes. Update the existing entries to reflect deprecation of root-level implementation functions.

**Step 2: Update rules section**

Add the DAG import direction rule:
- `dal` imports root only
- `operator` imports root + dal
- `service` imports root + operator + dal
- `cmd` imports service + root

**Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md with DAG architecture"
```

---

### Task 20: Regenerate wire and run final verification

**Step 1: Regenerate wire**

Run: `cd /Users/xiangjun/Documents/codebase/xj-m/xj-scripts/agentcli-go && wire ./service/`

**Step 2: Run full test suite**

Run: `go test ./...`
Expected: ALL PASS

**Step 3: Run vet**

Run: `go vet ./...`
Expected: no issues

**Step 4: Verify the binary still works**

Run: `go run ./cmd/agentcli --help`
Expected: shows usage

**Step 5: Test scaffold end-to-end**

Run:
```bash
tmp=$(mktemp -d)
go run ./cmd/agentcli new --name dag-test --module github.com/test/dag-test --dir "$tmp"
ls "$tmp/dag-test/service/" "$tmp/dag-test/dal/" "$tmp/dag-test/operator/"
rm -rf "$tmp"
```
Expected: DAG directories with expected files

**Step 6: Final commit**

```bash
git add -A
git commit -m "chore: final wire generation and verification"
```

---

Plan complete and saved to `docs/plans/2026-03-15-dag-architecture-implementation-plan.md`. Two execution options:

**1. Subagent-Driven (this session)** — I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** — Open new session with executing-plans, batch execution with checkpoints

Which approach?