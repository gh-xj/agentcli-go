package slotresource

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gh-xj/agentops/dal"
)

// WorktreeEntry represents a single git worktree entry from `git worktree list --porcelain`.
type WorktreeEntry struct {
	Path   string
	Branch string
	Head   string
	Bare   bool
}

// GitError wraps a failed git command with its arguments and output.
type GitError struct {
	Args   []string
	Output string
	Err    error
}

func (e *GitError) Error() string {
	return "git " + strings.Join(e.Args, " ") + ": " + e.Output
}

func (e *GitError) Unwrap() error { return e.Err }

// gitRun executes a git command in the given directory via dal.Executor.
func gitRun(exec dal.Executor, dir string, args ...string) (string, error) {
	out, err := exec.RunInDir(dir, "git", args...)
	if err != nil {
		return "", &GitError{Args: args, Output: strings.TrimSpace(out), Err: err}
	}
	return out, nil
}

// WorktreeList returns all worktree entries for the repository at repoDir.
func WorktreeList(exec dal.Executor, repoDir string) ([]WorktreeEntry, error) {
	out, err := gitRun(exec, repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreeList(out), nil
}

// WorktreeAdd creates a new worktree at path on a new branch.
func WorktreeAdd(exec dal.Executor, repoDir, path, branch string) error {
	_, err := gitRun(exec, repoDir, "worktree", "add", "-b", branch, path)
	return err
}

// WorktreeRemove removes the worktree at the given path. Uses --force because
// the application manages its own dirty-check logic (via IsDirtyExcluding) and
// the marker file is always present as an untracked file, which would otherwise
// cause git to refuse removal.
func WorktreeRemove(exec dal.Executor, repoDir, path string) error {
	_, err := gitRun(exec, repoDir, "worktree", "remove", "--force", path)
	return err
}

// IsDirtyExcluding returns true if the working tree has changes, ignoring files
// whose path (as reported by git status --porcelain) is in the excludeFiles set.
func IsDirtyExcluding(exec dal.Executor, dir string, excludeFiles map[string]bool) (bool, error) {
	out, err := gitRun(exec, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		// porcelain format: "XY filename" where XY is 2-char status + space
		file := strings.TrimSpace(line[2:])
		if excludeFiles != nil && excludeFiles[file] {
			continue
		}
		return true, nil
	}
	return false, nil
}

// CommitsBehind returns how many commits branch is behind baseBranch.
func CommitsBehind(exec dal.Executor, repoDir, branch, baseBranch string) (int, error) {
	out, err := gitRun(exec, repoDir, "rev-list", "--count", branch+".."+baseBranch)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, fmt.Errorf("parse rev-list count: %w", err)
	}
	return n, nil
}

// ListPrefixBranches returns all local branches matching "<prefix>/*".
func ListPrefixBranches(exec dal.Executor, repoDir, prefix string) ([]string, error) {
	pattern := prefix + "/*"
	out, err := gitRun(exec, repoDir, "branch", "--list", pattern, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// WorktreePrune runs git worktree prune to clean up stale entries.
func WorktreePrune(exec dal.Executor, repoDir string) error {
	_, err := gitRun(exec, repoDir, "worktree", "prune")
	return err
}

// BranchDelete deletes the named branch from the repository at repoDir.
func BranchDelete(exec dal.Executor, repoDir, branch string) error {
	_, err := gitRun(exec, repoDir, "branch", "-d", branch)
	return err
}

// FindRepoRoot walks up from dir looking for a .git directory. When run from
// inside a worktree (where .git is a file, not a directory), it follows the
// gitdir reference back to the main repo root.
func FindRepoRoot(fs dal.FileSystem, dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	dir = absDir

	for {
		gitPath := filepath.Join(dir, ".git")
		if fs.Exists(gitPath) {
			// Check if .git is a file (worktree) by trying to read it
			data, readErr := fs.ReadFile(gitPath)
			if readErr == nil && strings.HasPrefix(string(data), "gitdir: ") {
				// It's a worktree .git file — resolve to real repo
				resolved, err := resolveGitWorktree(fs, gitPath, data)
				if err == nil {
					return resolved, nil
				}
			}
			// It's a directory (real repo)
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found (searched up from %s)", absDir)
		}
		dir = parent
	}
}

// resolveGitWorktree reads a .git file containing "gitdir: <path>" and walks
// up from the referenced path to find the real repo root.
func resolveGitWorktree(fs dal.FileSystem, gitFile string, data []byte) (string, error) {
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("unexpected .git file content: %s", line)
	}
	gitdir := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(filepath.Dir(gitFile), gitdir)
	}

	// gitdir typically points to <repo>/.git/worktrees/<name>. Walk up to find the repo root.
	dir := filepath.Clean(gitdir)
	for {
		candidate := filepath.Dir(dir)
		if candidate == dir {
			break
		}
		dir = candidate
		gitPath := filepath.Join(dir, ".git")
		if fs.Exists(gitPath) {
			// Verify it's a directory (real repo), not another worktree .git file
			innerData, err := fs.ReadFile(gitPath)
			if err != nil || !strings.HasPrefix(string(innerData), "gitdir: ") {
				return dir, nil
			}
		}
	}
	return "", fmt.Errorf("could not resolve repo root from gitdir: %s", gitdir)
}

// parseWorktreeList parses the porcelain output of `git worktree list --porcelain`.
func parseWorktreeList(raw string) []WorktreeEntry {
	var entries []WorktreeEntry
	var cur WorktreeEntry
	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if cur.Path != "" {
				entries = append(entries, cur)
			}
			cur = WorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			cur.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "bare":
			cur.Bare = true
		}
	}
	if cur.Path != "" {
		entries = append(entries, cur)
	}
	return entries
}

// slotInfo holds parsed information about a git worktree slot.
// Used internally by slot resource operations.
type slotInfo struct {
	Name   string
	Path   string
	Branch string
}

// createWorktree creates a git worktree at ../worktrees/<project>-<name>
// with a branch named <prefix>/<name>. Writes a marker file.
func createWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string, cfg *SlotConfig) (slotInfo, error) {
	paths := cfg.SlotPaths(projectDir, name)
	worktreePath := paths.WorktreePath
	branchName := paths.Branch

	// Check if worktree path already exists
	if fs.Exists(worktreePath) {
		return slotInfo{}, fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Check if branch already exists
	_, err := gitRun(exec, projectDir, "rev-parse", "--verify", branchName)
	if err == nil {
		return slotInfo{}, fmt.Errorf("branch %q already exists", branchName)
	}

	// Ensure parent directory exists
	worktreesDir := filepath.Dir(worktreePath)
	if err := fs.EnsureDir(worktreesDir); err != nil {
		return slotInfo{}, fmt.Errorf("create worktrees dir: %w", err)
	}

	// Create worktree with new branch
	if err := WorktreeAdd(exec, projectDir, worktreePath, branchName); err != nil {
		return slotInfo{}, fmt.Errorf("create worktree: %w", err)
	}

	// Write marker file
	markerPath := paths.MarkerPath
	if err := fs.WriteFile(markerPath, []byte(name), 0o644); err != nil {
		return slotInfo{}, fmt.Errorf("write marker file: %w", err)
	}

	// Configure git user if not set
	emailOut, _ := exec.RunInDir(worktreePath, "git", "config", "user.email")
	if strings.TrimSpace(emailOut) == "" {
		exec.RunInDir(worktreePath, "git", "config", "user.email", "agentcli@local")
		exec.RunInDir(worktreePath, "git", "config", "user.name", "agentcli")
	}

	// Stage and commit the marker file
	if _, err := gitRun(exec, worktreePath, "add", cfg.MarkerFile); err != nil {
		return slotInfo{}, fmt.Errorf("git add marker: %w", err)
	}
	if _, err := gitRun(exec, worktreePath, "commit", "-m", fmt.Sprintf("slot: init %s", name)); err != nil {
		return slotInfo{}, fmt.Errorf("git commit marker: %w", err)
	}

	return slotInfo{
		Name:   name,
		Path:   worktreePath,
		Branch: branchName,
	}, nil
}

// listWorktrees returns all worktrees matching the project prefix that have
// a marker file. Uses the porcelain worktree list parser.
func listWorktrees(exec dal.Executor, fs dal.FileSystem, projectDir string, cfg *SlotConfig) ([]slotInfo, error) {
	entries, err := WorktreeList(exec, projectDir)
	if err != nil {
		return nil, err
	}

	prefix := cfg.WorktreePrefix + "-"
	var slots []slotInfo
	for _, entry := range entries {
		base := filepath.Base(entry.Path)
		if !strings.HasPrefix(base, prefix) {
			continue
		}
		markerPath := filepath.Join(entry.Path, cfg.MarkerFile)
		data, err := fs.ReadFile(markerPath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		slots = append(slots, slotInfo{
			Name:   name,
			Path:   entry.Path,
			Branch: entry.Branch,
		})
	}
	return slots, nil
}

// removeWorktree checks for uncommitted changes (excluding the marker file),
// then removes the worktree and deletes the branch (best-effort).
func removeWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string, cfg *SlotConfig) error {
	paths := cfg.SlotPaths(projectDir, name)
	worktreePath := paths.WorktreePath
	branchName := paths.Branch

	if !fs.Exists(worktreePath) {
		return fmt.Errorf("slot %q not found at %s", name, worktreePath)
	}

	// Check for uncommitted changes (excluding marker file)
	dirty, err := IsDirtyExcluding(exec, worktreePath, cfg.MarkerFiles())
	if err != nil {
		return fmt.Errorf("check dirty: %w", err)
	}
	if dirty {
		return fmt.Errorf("slot %q has uncommitted changes; commit or stash first", name)
	}

	// Remove worktree (--force handles the marker file)
	if err := WorktreeRemove(exec, projectDir, worktreePath); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	// Best-effort branch delete
	_ = BranchDelete(exec, projectDir, branchName)

	return nil
}

// syncWorktree fetches latest base branch and rebases the slot branch onto it.
func syncWorktree(exec dal.Executor, fs dal.FileSystem, projectDir, name string, cfg *SlotConfig) error {
	paths := cfg.SlotPaths(projectDir, name)
	worktreePath := paths.WorktreePath

	if !fs.Exists(worktreePath) {
		return fmt.Errorf("slot %q not found at %s", name, worktreePath)
	}

	// Fetch latest base branch (ignore errors for local-only repos)
	exec.RunInDir(projectDir, "git", "fetch", "origin", cfg.BaseBranch)

	// Rebase
	out, err := exec.RunInDir(worktreePath, "git", "rebase", "origin/"+cfg.BaseBranch)
	if err != nil {
		// Abort failed rebase
		exec.RunInDir(worktreePath, "git", "rebase", "--abort")
		return fmt.Errorf("rebase failed (aborted): %s: %w", strings.TrimSpace(out), err)
	}

	return nil
}

// ReadMarker reads the marker file from a directory. Returns "" if the file does not exist.
func ReadMarker(fs dal.FileSystem, path, markerFile string) string {
	data, err := fs.ReadFile(filepath.Join(path, markerFile))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	return s
}
