package slotresource

import (
	"fmt"
	"path/filepath"

	"github.com/gh-xj/agentops/dal"
	"gopkg.in/yaml.v3"
)

// SlotConfig represents the parsed .agentops/slot.yaml configuration.
type SlotConfig struct {
	Slots          []string `yaml:"slots"`           // pre-declared names (optional)
	BaseBranch     string   `yaml:"base_branch"`     // default: main
	WorktreePrefix string   `yaml:"worktree_prefix"` // default: repo dirname
	BranchPrefix   string   `yaml:"branch_prefix"`   // default: slot
	MarkerFile     string   `yaml:"marker_file"`     // default: .slot
}

// SlotPaths holds the computed paths for a single worktree slot.
type SlotPaths struct {
	Name         string
	WorktreePath string
	Branch       string
	MarkerPath   string
}

// LoadSlotConfig reads and parses .agentops/slot.yaml. If the file does not
// exist, sensible defaults are returned. The repoRoot is used to derive the
// default worktree prefix from the repository directory name.
func LoadSlotConfig(fs dal.FileSystem, agentopsDir string, repoRoot string) (*SlotConfig, error) {
	cfg := &SlotConfig{}

	path := filepath.Join(agentopsDir, "slot.yaml")
	data, err := fs.ReadFile(path)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse slot.yaml: %w", err)
		}
	}
	// If file doesn't exist, use all defaults

	// Apply defaults
	if cfg.WorktreePrefix == "" {
		cfg.WorktreePrefix = filepath.Base(repoRoot)
	}
	if cfg.BranchPrefix == "" {
		cfg.BranchPrefix = "slot"
	}
	if cfg.MarkerFile == "" {
		cfg.MarkerFile = ".slot"
	}
	if cfg.BaseBranch == "" {
		cfg.BaseBranch = "main"
	}

	return cfg, nil
}

// ValidateSlotName checks whether name is a valid slot in this configuration.
func (c *SlotConfig) ValidateSlotName(name string) error {
	if name == "" {
		return fmt.Errorf("slot name must not be empty")
	}
	for _, s := range c.Slots {
		if s == name {
			return nil
		}
	}
	return fmt.Errorf("unknown slot %q; valid slots: %v", name, c.Slots)
}

// SlotPaths computes the worktree path, branch name, and marker path for a slot.
// The worktree is placed as a sibling to the repo root under a "worktrees" directory.
func (c *SlotConfig) SlotPaths(repoRoot, name string) SlotPaths {
	wtDir := filepath.Join(filepath.Dir(repoRoot), "worktrees", c.WorktreePrefix+"-"+name)
	return SlotPaths{
		Name:         name,
		WorktreePath: wtDir,
		Branch:       c.BranchPrefix + "/" + name,
		MarkerPath:   filepath.Join(wtDir, c.MarkerFile),
	}
}

// MarkerFiles returns a map of marker file names for dirty-check exclusion.
func (c *SlotConfig) MarkerFiles() map[string]bool {
	return map[string]bool{
		c.MarkerFile: true,
	}
}
