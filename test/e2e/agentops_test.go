package e2e

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	// This file lives at test/e2e/agentops_test.go, so repo root is two levels up.
	here, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs cwd: %v", err)
	}
	root := filepath.Join(here, "..", "..")
	root, err = filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs root: %v", err)
	}
	return root
}

// buildBinary compiles the agentops binary into a temp directory and returns
// the path to the executable.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "agentops")
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/agentops/")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %s\n%s", err, out)
	}
	return binary
}

// runCmd executes the agentops binary with the given arguments and returns
// the combined stdout+stderr output and the process exit code.
func runCmd(t *testing.T, binary string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error (not ExitError): %v", err)
		}
	}
	return string(out), exitCode
}

// runCmdInDir is like runCmd but sets the working directory for the process.
func runCmdInDir(t *testing.T, binary, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec error (not ExitError): %v", err)
		}
	}
	return string(out), exitCode
}

// initProject creates a temp directory, runs git init, and runs agentops init
// with in-repo storage so that case operations work within the single directory.
func initProject(t *testing.T, binary string) string {
	t.Helper()
	dir := t.TempDir()

	// git init so strategy discovery has a valid project root.
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = dir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s\n%s", err, out)
	}

	// agentops init --dir <dir>
	out, code := runCmd(t, binary, "init", "--dir", dir)
	if code != 0 {
		t.Fatalf("agentops init failed (exit %d): %s", code, out)
	}

	// Patch storage.yaml to use in-repo backend so case commands store data
	// inside the project directory rather than a sibling repo.
	storageYAML := filepath.Join(dir, ".agentops", "storage.yaml")
	if err := os.WriteFile(storageYAML, []byte("backend: in-repo\n"), 0o644); err != nil {
		t.Fatalf("write storage.yaml: %v", err)
	}

	return dir
}

func TestVersion(t *testing.T) {
	binary := buildBinary(t)
	out, code := runCmd(t, binary, "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", code, out)
	}
	if !strings.Contains(strings.ToLower(out), "agentops") {
		t.Errorf("expected output to contain 'agentops', got: %s", out)
	}
}

func TestHelp(t *testing.T) {
	binary := buildBinary(t)
	out, code := runCmd(t, binary, "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; output: %s", code, out)
	}
	for _, keyword := range []string{"case", "slot", "project"} {
		if !strings.Contains(out, keyword) {
			t.Errorf("expected help output to contain %q, got:\n%s", keyword, out)
		}
	}
}

func TestInitAndDoctor(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// git init
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = dir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s\n%s", err, out)
	}

	// agentops init --dir <dir>
	out, code := runCmd(t, binary, "init", "--dir", dir)
	if code != 0 {
		t.Fatalf("agentops init failed (exit %d): %s", code, out)
	}

	// Verify .agentops/ directory was created.
	agentopsDir := filepath.Join(dir, ".agentops")
	info, err := os.Stat(agentopsDir)
	if err != nil {
		t.Fatalf(".agentops/ not found after init: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf(".agentops/ exists but is not a directory")
	}

	// Verify key files exist.
	for _, name := range []string{"storage.yaml", "transitions.yaml"} {
		p := filepath.Join(agentopsDir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}

	// agentops doctor --dir <dir> (run from within the dir so strategy discovers .agentops/)
	out, code = runCmdInDir(t, binary, dir, "doctor", "--dir", dir)
	if code != 0 {
		t.Fatalf("agentops doctor failed (exit %d): %s", code, out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("expected doctor output to contain 'ok', got: %s", out)
	}
}

func TestCaseWorkflow(t *testing.T) {
	binary := buildBinary(t)
	dir := initProject(t, binary)

	// Create a case.
	out, code := runCmdInDir(t, binary, dir, "case", "create", "fix-bug")
	if code != 0 {
		t.Fatalf("case create failed (exit %d): %s", code, out)
	}
	if !strings.Contains(out, "fix-bug") {
		t.Errorf("expected create output to mention 'fix-bug', got: %s", out)
	}

	// Extract the case ID from the create output.
	// The ID follows the pattern CASE-YYYYMMDD-fix-bug
	caseIDPattern := regexp.MustCompile(`CASE-\d{8}-fix-bug`)
	match := caseIDPattern.FindString(out)
	if match == "" {
		t.Fatalf("could not extract case ID from create output: %s", out)
	}
	caseID := match

	// List cases — should see exactly 1 case.
	out, code = runCmdInDir(t, binary, dir, "case", "list")
	if code != 0 {
		t.Fatalf("case list failed (exit %d): %s", code, out)
	}
	if !strings.Contains(out, caseID) {
		t.Errorf("expected list output to contain %q, got:\n%s", caseID, out)
	}
	// Count data rows (non-header lines containing CASE-).
	lines := strings.Split(strings.TrimSpace(out), "\n")
	caseLines := 0
	for _, line := range lines {
		if strings.Contains(line, "CASE-") {
			caseLines++
		}
	}
	if caseLines != 1 {
		t.Errorf("expected 1 case in list, found %d; output:\n%s", caseLines, out)
	}

	// Get the specific case.
	out, code = runCmdInDir(t, binary, dir, "case", "get", caseID)
	if code != 0 {
		t.Fatalf("case get failed (exit %d): %s", code, out)
	}
	if !strings.Contains(out, caseID) {
		t.Errorf("expected get output to contain %q, got:\n%s", caseID, out)
	}
}

func TestInitIdempotent(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// git init
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = dir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s\n%s", err, out)
	}

	// Run init twice.
	for i := 0; i < 2; i++ {
		out, code := runCmd(t, binary, "init", "--dir", dir)
		if code != 0 {
			t.Fatalf("agentops init (run %d) failed (exit %d): %s", i+1, code, out)
		}
	}

	// Verify .agentops/ still exists and is valid.
	agentopsDir := filepath.Join(dir, ".agentops")
	info, err := os.Stat(agentopsDir)
	if err != nil {
		t.Fatalf(".agentops/ not found after double init: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf(".agentops/ is not a directory after double init")
	}

	// Doctor should still pass.
	out, code := runCmdInDir(t, binary, dir, "doctor", "--dir", dir)
	if code != 0 {
		t.Fatalf("doctor failed after double init (exit %d): %s", code, out)
	}
}

func TestDoctorWithoutInit(t *testing.T) {
	binary := buildBinary(t)
	dir := t.TempDir()

	// No git init, no agentops init — doctor should fail.
	out, code := runCmdInDir(t, binary, dir, "doctor", "--dir", dir)
	if code == 0 {
		t.Fatalf("expected non-zero exit from doctor without init, got exit 0; output: %s", out)
	}
	_ = fmt.Sprintf("doctor exited %d as expected: %s", code, out)
}
