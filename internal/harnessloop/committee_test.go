package harnessloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRoleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roles.json")
	content := `{"planner":{"strategy":"external","command":"echo planner"},"fixer":{"strategy":"builtin"},"judger":{"strategy":"builtin"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write role config: %v", err)
	}
	cfg, err := loadRoleConfig(path)
	if err != nil {
		t.Fatalf("load role config: %v", err)
	}
	if cfg.Planner.Command == "" || cfg.Planner.Strategy != "external" {
		t.Fatalf("unexpected planner config: %+v", cfg.Planner)
	}
}

func TestStrategyOrBuiltin(t *testing.T) {
	if got := strategyOrBuiltin(RoleSpec{}); got != "builtin" {
		t.Fatalf("unexpected strategy: %s", got)
	}
	if got := strategyOrBuiltin(RoleSpec{Command: "echo hi"}); got != "external" {
		t.Fatalf("unexpected strategy from command: %s", got)
	}
	if got := strategyOrBuiltin(RoleSpec{Strategy: "custom"}); got != "custom" {
		t.Fatalf("unexpected explicit strategy: %s", got)
	}
}
