package slotresource

import (
	"fmt"
	"regexp"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/resource"
)

var slotNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// SlotResource implements Resource, Deleter, and Syncer for git worktree slots.
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
func (s *SlotResource) projectDir(ctx *agentcli.AppContext) (string, error) {
	if v, ok := ctx.Values["project_dir"]; ok {
		if dir, ok := v.(string); ok && dir != "" {
			return dir, nil
		}
	}
	out, err := s.exec.Run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("detect project dir: %w", err)
	}
	dir := trimOutput(out)
	if dir == "" {
		return "", fmt.Errorf("not inside a git repository")
	}
	return dir, nil
}

// Create validates the name and creates a worktree slot.
func (s *SlotResource) Create(ctx *agentcli.AppContext, slug string, opts map[string]string) (*resource.Record, error) {
	if !slotNamePattern.MatchString(slug) {
		return nil, fmt.Errorf("invalid slot name %q: must match ^[a-z][a-z0-9-]*$", slug)
	}

	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return nil, err
	}

	info, err := createWorktree(s.exec, s.fs, projectDir, slug)
	if err != nil {
		return nil, fmt.Errorf("create slot %q: %w", slug, err)
	}

	return infoToRecord(info), nil
}

// List returns all slots for the project.
func (s *SlotResource) List(ctx *agentcli.AppContext, filter resource.Filter) ([]resource.Record, error) {
	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return nil, err
	}

	infos, err := listWorktrees(s.exec, s.fs, projectDir)
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
func (s *SlotResource) Get(ctx *agentcli.AppContext, id string) (*resource.Record, error) {
	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return nil, err
	}

	infos, err := listWorktrees(s.exec, s.fs, projectDir)
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
func (s *SlotResource) Delete(ctx *agentcli.AppContext, id string) error {
	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return err
	}
	return removeWorktree(s.exec, s.fs, projectDir, id)
}

// Sync rebases the slot branch onto origin/main.
func (s *SlotResource) Sync(ctx *agentcli.AppContext, id string) error {
	projectDir, err := s.projectDir(ctx)
	if err != nil {
		return err
	}
	return syncWorktree(s.exec, s.fs, projectDir, id)
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

// trimOutput trims whitespace from command output.
func trimOutput(s string) string {
	// Trim all trailing and leading whitespace including newlines
	result := ""
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	result = s[start:end]
	return result
}
