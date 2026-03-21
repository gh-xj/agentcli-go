package slotresource

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gh-xj/agentcli-go/dal"
)

// slotInfo holds parsed information about a git worktree slot.
type slotInfo struct {
	Name   string
	Path   string
	Branch string
}

// createWorktree creates a git worktree at ../worktrees/<project>-<name>
// with a branch named slot/<name>. Writes a .slot marker file.
func createWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string) (slotInfo, error) {
	projectName := filepath.Base(projectDir)
	worktreesDir := filepath.Join(filepath.Dir(projectDir), "worktrees")
	worktreePath := filepath.Join(worktreesDir, projectName+"-"+name)
	branchName := "slot/" + name

	// Check if worktree path already exists
	if fs.Exists(worktreePath) {
		return slotInfo{}, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Check if branch already exists
	_, err := exec.RunInDir(projectDir, "git", "rev-parse", "--verify", branchName)
	if err == nil {
		return slotInfo{}, fmt.Errorf("branch %q already exists", branchName)
	}

	// Ensure parent directory exists
	if err := fs.EnsureDir(worktreesDir); err != nil {
		return slotInfo{}, fmt.Errorf("create worktrees dir: %w", err)
	}

	// Create worktree with new branch
	out, err := exec.RunInDir(projectDir, "git", "worktree", "add", "-b", branchName, worktreePath)
	if err != nil {
		return slotInfo{}, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(out), err)
	}

	// Write .slot marker
	slotFile := filepath.Join(worktreePath, ".slot")
	if err := fs.WriteFile(slotFile, []byte(name), 0o644); err != nil {
		return slotInfo{}, fmt.Errorf("write .slot marker: %w", err)
	}

	// Configure git user if not set
	emailOut, _ := exec.RunInDir(worktreePath, "git", "config", "user.email")
	if strings.TrimSpace(emailOut) == "" {
		exec.RunInDir(worktreePath, "git", "config", "user.email", "agentcli@local")
		exec.RunInDir(worktreePath, "git", "config", "user.name", "agentcli")
	}

	// Stage and commit the .slot marker
	if out, err := exec.RunInDir(worktreePath, "git", "add", ".slot"); err != nil {
		return slotInfo{}, fmt.Errorf("git add .slot: %s: %w", strings.TrimSpace(out), err)
	}
	if out, err := exec.RunInDir(worktreePath, "git", "commit", "-m", fmt.Sprintf("slot: init %s", name)); err != nil {
		return slotInfo{}, fmt.Errorf("git commit .slot: %s: %w", strings.TrimSpace(out), err)
	}

	return slotInfo{
		Name:   name,
		Path:   worktreePath,
		Branch: branchName,
	}, nil
}

// listWorktrees returns all worktrees matching the project prefix that have
// a .slot marker. Parses output of `git worktree list --porcelain`.
func listWorktrees(exec dal.Executor, fs dal.FileSystem, projectDir string) ([]slotInfo, error) {
	projectName := filepath.Base(projectDir)
	prefix := projectName + "-"

	out, err := exec.RunInDir(projectDir, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var slots []slotInfo
	var current slotInfo
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			current = slotInfo{Path: strings.TrimPrefix(line, "worktree ")}
		} else if strings.HasPrefix(line, "branch refs/heads/") {
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == "" && current.Path != "" {
			base := filepath.Base(current.Path)
			if strings.HasPrefix(base, prefix) {
				slotMarker := filepath.Join(current.Path, ".slot")
				data, err := fs.ReadFile(slotMarker)
				if err == nil {
					current.Name = strings.TrimSpace(string(data))
					slots = append(slots, current)
				}
			}
			current = slotInfo{}
		}
	}
	return slots, nil
}

// removeWorktree checks for uncommitted changes, then removes the worktree
// and deletes the branch (best-effort).
func removeWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string) error {
	projectName := filepath.Base(projectDir)
	worktreePath := filepath.Join(filepath.Dir(projectDir), "worktrees", projectName+"-"+name)
	branchName := "slot/" + name

	if !fs.Exists(worktreePath) {
		return fmt.Errorf("slot %q not found at %s", name, worktreePath)
	}

	// Check for uncommitted changes
	statusOut, err := exec.RunInDir(worktreePath, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(statusOut) != "" {
		return fmt.Errorf("slot %q has uncommitted changes; commit or stash first", name)
	}

	// Remove worktree
	out, err := exec.RunInDir(projectDir, "git", "worktree", "remove", worktreePath)
	if err != nil {
		return fmt.Errorf("git worktree remove: %s: %w", strings.TrimSpace(out), err)
	}

	// Best-effort branch delete
	exec.RunInDir(projectDir, "git", "branch", "-d", branchName)

	return nil
}

// syncWorktree fetches latest main and rebases the slot branch onto origin/main.
func syncWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string) error {
	projectName := filepath.Base(projectDir)
	worktreePath := filepath.Join(filepath.Dir(projectDir), "worktrees", projectName+"-"+name)

	if !fs.Exists(worktreePath) {
		return fmt.Errorf("slot %q not found at %s", name, worktreePath)
	}

	// Fetch latest main (ignore errors for local-only repos)
	exec.RunInDir(projectDir, "git", "fetch", "origin", "main")

	// Rebase
	out, err := exec.RunInDir(worktreePath, "git", "rebase", "origin/main")
	if err != nil {
		// Abort failed rebase
		exec.RunInDir(worktreePath, "git", "rebase", "--abort")
		return fmt.Errorf("rebase failed (aborted): %s: %w", strings.TrimSpace(out), err)
	}

	return nil
}
