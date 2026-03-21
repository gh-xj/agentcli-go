package slotresource

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	agentops "github.com/gh-xj/agentops"
	"github.com/gh-xj/agentops/dal"
	"github.com/gh-xj/agentops/resource"
)

var slotNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// SlotResource implements Resource, Deleter, Syncer, Doctor, and Pruner for git worktree slots.
type SlotResource struct {
	fs   dal.FileSystem
	exec dal.Executor
}

// New creates a SlotResource with the given filesystem and executor.
func New(fs dal.FileSystem, exec dal.Executor) *SlotResource {
	return &SlotResource{fs: fs, exec: exec}
}

// Schema returns the resource schema for slots.
func (s *SlotResource) Schema() resource.ResourceSchema {
	return resource.ResourceSchema{
		Kind:        "slot",
		Description: "Git worktree-based development slot",
		Fields: []resource.FieldDef{
			{Name: "name", Type: "string", Required: true},
			{Name: "path", Type: "string", Required: true},
			{Name: "branch", Type: "string", Required: true},
		},
		CreateArgs: []resource.ArgDef{
			{Name: "name", Description: "Slot name (lowercase alphanumeric with hyphens)", Required: true},
		},
	}
}

// projectDir resolves the project directory from the AppContext or detects it
// from the current git repo root.
func (s *SlotResource) projectDir(ctx *agentops.AppContext) (string, error) {
	if v, ok := ctx.Values["project_dir"]; ok {
		if dir, ok := v.(string); ok && dir != "" {
			return dir, nil
		}
	}
	out, err := s.exec.Run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("detect project dir: %w", err)
	}
	dir := strings.TrimSpace(out)
	if dir == "" {
		return "", fmt.Errorf("not inside a git repository")
	}
	return dir, nil
}

// loadConfig resolves the project directory and loads the slot config.
// The project directory is resolved to its real path (symlinks evaluated) to
// ensure consistent path comparisons with git output.
func (s *SlotResource) loadConfig(ctx *agentops.AppContext) (string, *SlotConfig, error) {
	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return "", nil, err
	}
	// Resolve symlinks (e.g., /var -> /private/var on macOS) so paths
	// from cfg.SlotPaths match those returned by git worktree list.
	if resolved, err := os.Readlink(projectDir); err == nil {
		projectDir = resolved
	}
	if resolved, err := filepath.EvalSymlinks(projectDir); err == nil {
		projectDir = resolved
	}
	agentopsDir := filepath.Join(projectDir, ".agentops")
	cfg, err := LoadSlotConfig(s.fs, agentopsDir, projectDir)
	if err != nil {
		return "", nil, fmt.Errorf("load slot config: %w", err)
	}
	return projectDir, cfg, nil
}

// Create validates the name and creates a worktree slot.
func (s *SlotResource) Create(ctx *agentops.AppContext, slug string, opts map[string]string) (*resource.Record, error) {
	if !slotNamePattern.MatchString(slug) {
		return nil, fmt.Errorf("invalid slot name %q: must match ^[a-z][a-z0-9-]*$", slug)
	}

	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	// If config has an explicit slots list, validate against it
	if len(cfg.Slots) > 0 {
		if err := cfg.ValidateSlotName(slug); err != nil {
			return nil, err
		}
	}

	info, err := createWorktree(s.exec, s.fs, projectDir, slug, cfg)
	if err != nil {
		return nil, fmt.Errorf("create slot %q: %w", slug, err)
	}

	return infoToRecord(info), nil
}

// List returns all slots for the project.
func (s *SlotResource) List(ctx *agentops.AppContext, filter resource.Filter) ([]resource.Record, error) {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	infos, err := listWorktrees(s.exec, s.fs, projectDir, cfg)
	if err != nil {
		return nil, err
	}

	records := make([]resource.Record, 0, len(infos))
	for _, info := range infos {
		records = append(records, *infoToRecord(info))
	}
	return records, nil
}

// Get returns a single slot by name.
func (s *SlotResource) Get(ctx *agentops.AppContext, id string) (*resource.Record, error) {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	infos, err := listWorktrees(s.exec, s.fs, projectDir, cfg)
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		if info.Name == id {
			return infoToRecord(info), nil
		}
	}
	return nil, fmt.Errorf("slot %q not found", id)
}

// Delete removes a slot worktree after checking for uncommitted changes.
func (s *SlotResource) Delete(ctx *agentops.AppContext, id string) error {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	return removeWorktree(s.exec, s.fs, projectDir, id, cfg)
}

// Sync rebases the slot branch onto origin/<base_branch>.
func (s *SlotResource) Sync(ctx *agentops.AppContext, id string) error {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	return syncWorktree(s.exec, s.fs, projectDir, id, cfg)
}

// Doctor runs health checks on all active slot worktrees.
func (s *SlotResource) Doctor(ctx *agentops.AppContext) ([]resource.DoctorCheck, error) {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	entries, err := WorktreeList(s.exec, projectDir)
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	// Build lookup maps
	wtByPath := make(map[string]WorktreeEntry)
	wtByBranch := make(map[string]WorktreeEntry)
	for _, e := range entries {
		wtByPath[e.Path] = e
		if e.Branch != "" {
			wtByBranch[e.Branch] = e
		}
	}

	// Collect slot names to check: either from config.Slots or from discovered worktrees.
	// When discovering, we use the worktree path prefix rather than the marker file,
	// since a missing marker is itself a health issue we want to detect.
	slotNames := cfg.Slots
	if len(slotNames) == 0 {
		prefix := cfg.WorktreePrefix + "-"
		for _, entry := range entries {
			base := filepath.Base(entry.Path)
			if strings.HasPrefix(base, prefix) {
				name := strings.TrimPrefix(base, prefix)
				slotNames = append(slotNames, name)
			}
		}
	}

	excludes := cfg.MarkerFiles()
	var results []resource.DoctorCheck

	// Per-slot checks
	for _, name := range slotNames {
		slot := cfg.SlotPaths(projectDir, name)

		if _, exists := wtByPath[slot.WorktreePath]; !exists {
			continue // Slot not active
		}

		slotHasIssue := false

		// Check 1: Missing/wrong marker
		marker := ReadMarker(s.fs, slot.WorktreePath, cfg.MarkerFile)
		if marker == "" {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "missing_marker",
				Message:  "marker file missing",
				Severity: "warn",
			})
		} else if marker != name {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "wrong_marker",
				Message:  fmt.Sprintf("marker contains %q, expected %q", marker, name),
				Severity: "warn",
			})
		}

		// Check 2: Dirty worktree
		dirty, dirtyErr := IsDirtyExcluding(s.exec, slot.WorktreePath, excludes)
		if dirtyErr != nil {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "check_error",
				Message:  fmt.Sprintf("cannot check dirty status: %v", dirtyErr),
				Severity: "warn",
			})
		} else if dirty {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "dirty",
				Message:  "uncommitted changes",
				Severity: "warn",
			})
		}

		// Check 3: Behind base branch
		behind, behindErr := CommitsBehind(s.exec, projectDir, slot.Branch, cfg.BaseBranch)
		if behindErr != nil {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "check_error",
				Message:  fmt.Sprintf("cannot check behind status: %v", behindErr),
				Severity: "warn",
			})
		} else if behind > 0 {
			slotHasIssue = true
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "behind",
				Message:  fmt.Sprintf("%d commits behind %s", behind, cfg.BaseBranch),
				Severity: "warn",
			})
		}

		if !slotHasIssue {
			results = append(results, resource.DoctorCheck{
				Name:     name,
				Status:   "ok",
				Message:  "clean, up to date",
				Severity: "ok",
			})
		}
	}

	// Check 4: Stale branches (branches matching prefix/* with no worktree)
	branches, branchErr := ListPrefixBranches(s.exec, projectDir, cfg.BranchPrefix)
	if branchErr == nil {
		for _, branch := range branches {
			if _, hasWT := wtByBranch[branch]; !hasWT {
				results = append(results, resource.DoctorCheck{
					Name:     branch,
					Status:   "stale_branch",
					Message:  "stale branch (no worktree)",
					Severity: "err",
				})
			}
		}
	}

	// Check 5: Orphaned worktree directories
	worktreesDir := filepath.Join(filepath.Dir(projectDir), "worktrees")
	dirEntries, readErr := s.fs.ReadDir(worktreesDir)
	if readErr == nil {
		prefix := cfg.WorktreePrefix + "-"
		for _, de := range dirEntries {
			if !de.IsDir {
				continue
			}
			dirName := de.Name
			if !strings.HasPrefix(dirName, prefix) {
				continue
			}
			slotName := strings.TrimPrefix(dirName, prefix)
			expectedBranch := cfg.BranchPrefix + "/" + slotName
			if _, hasBranch := wtByBranch[expectedBranch]; !hasBranch {
				orphanPath := filepath.Join(worktreesDir, dirName)
				results = append(results, resource.DoctorCheck{
					Name:     orphanPath,
					Status:   "orphaned",
					Message:  "orphaned (no slot branch)",
					Severity: "err",
				})
			}
		}
	}

	return results, nil
}

// Prune removes clean stale worktrees. Dry-run by default (confirm=false).
func (s *SlotResource) Prune(ctx *agentops.AppContext, confirm bool) ([]resource.PruneResult, error) {
	projectDir, cfg, err := s.loadConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Step 1: Run git worktree prune to clean stale git entries
	if err := WorktreePrune(s.exec, projectDir); err != nil {
		return nil, fmt.Errorf("worktree prune: %w", err)
	}

	// Step 2: Discover active slot worktrees
	entries, err := WorktreeList(s.exec, projectDir)
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	wtByPath := make(map[string]WorktreeEntry)
	for _, e := range entries {
		wtByPath[e.Path] = e
	}

	slotNames := cfg.Slots
	if len(slotNames) == 0 {
		prefix := cfg.WorktreePrefix + "-"
		for _, entry := range entries {
			base := filepath.Base(entry.Path)
			if strings.HasPrefix(base, prefix) {
				name := strings.TrimPrefix(base, prefix)
				slotNames = append(slotNames, name)
			}
		}
	}

	excludes := cfg.MarkerFiles()
	var results []resource.PruneResult

	for _, name := range slotNames {
		slot := cfg.SlotPaths(projectDir, name)

		if _, exists := wtByPath[slot.WorktreePath]; !exists {
			continue // Slot not active, nothing to prune
		}

		dirty, dirtyErr := IsDirtyExcluding(s.exec, slot.WorktreePath, excludes)
		if dirtyErr != nil {
			results = append(results, resource.PruneResult{
				Name:   name,
				Path:   slot.WorktreePath,
				Action: "skipped",
				Reason: fmt.Sprintf("cannot check dirty status: %v", dirtyErr),
			})
			continue
		}

		if dirty {
			results = append(results, resource.PruneResult{
				Name:   name,
				Path:   slot.WorktreePath,
				Action: "skipped",
				Reason: "dirty (uncommitted changes)",
			})
			continue
		}

		// Clean slot — remove or report
		if confirm {
			removeErr := WorktreeRemove(s.exec, projectDir, slot.WorktreePath)
			if removeErr != nil {
				results = append(results, resource.PruneResult{
					Name:   name,
					Path:   slot.WorktreePath,
					Action: "skipped",
					Reason: fmt.Sprintf("remove failed: %v", removeErr),
				})
				continue
			}
			_ = BranchDelete(s.exec, projectDir, slot.Branch)
			results = append(results, resource.PruneResult{
				Name:   name,
				Path:   slot.WorktreePath,
				Action: "removed",
			})
		} else {
			results = append(results, resource.PruneResult{
				Name:   name,
				Path:   slot.WorktreePath,
				Action: "would_remove",
			})
		}
	}

	return results, nil
}

// infoToRecord converts a slotInfo to a resource.Record.
func infoToRecord(info slotInfo) *resource.Record {
	return &resource.Record{
		Kind: "slot",
		ID:   info.Name,
		Fields: map[string]any{
			"name":   info.Name,
			"path":   info.Path,
			"branch": info.Branch,
		},
		RawPath: info.Path,
	}
}
