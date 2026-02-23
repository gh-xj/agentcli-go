package harnessloop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoopDoctorReportsLabReadiness(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1\nwhich agentcli\nagentcli --version\nagentcli --help\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "prompts"), 0755); err != nil {
		t.Fatalf("mkdir prompts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "prompts", "agentcli-onboarding.prompt.md"), []byte("go install github.com/gh-xj/agentcli-go/cmd/agentcli@v0.2.1\nwhich agentcli\nagentcli --version\nagentcli --help\n"), 0644); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".docs", "onboarding-loop", "runs", "x", "iter-01"), 0755); err != nil {
		t.Fatalf("mkdir iter: %v", err)
	}
	r := LoopDoctor(repo)
	if !r.LeanReady {
		t.Fatalf("expected lean ready")
	}
	if !r.LabFeaturesReady {
		t.Fatalf("expected lab ready")
	}
}
