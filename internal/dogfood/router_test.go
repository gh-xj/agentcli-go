package dogfood

import "testing"

func TestResolveRepoUsesOverrideFirst(t *testing.T) {
	r := Router{MinConfidence: 0.75}

	res := r.Resolve(RouteInput{
		OverrideRepo: "gh-xj/agentcli-go",
		CWD:          "/tmp/x",
		GitRemote:    "git@github.com:other/repo.git",
	})

	if res.Repo != "gh-xj/agentcli-go" {
		t.Fatalf("expected override repo, got %q", res.Repo)
	}
	if res.Confidence != 1.0 {
		t.Fatalf("expected confidence=1.0, got %v", res.Confidence)
	}
	if res.Reason != "manual_override" {
		t.Fatalf("expected manual_override reason, got %q", res.Reason)
	}
	if res.Pending {
		t.Fatalf("override route should not be pending")
	}
}

func TestResolveRepoMarksPendingWhenSignalIsWeak(t *testing.T) {
	r := Router{MinConfidence: 0.75}

	res := r.Resolve(RouteInput{
		CWD: "/tmp/worktree/my-repo",
	})

	if res.Repo != "my-repo" {
		t.Fatalf("expected cwd repo guess, got %q", res.Repo)
	}
	if !res.Pending {
		t.Fatalf("expected pending for low confidence route")
	}
	if res.Confidence >= r.MinConfidence {
		t.Fatalf("expected low confidence, got %v", res.Confidence)
	}
	if res.Reason != "cwd_guess" {
		t.Fatalf("expected cwd_guess reason, got %q", res.Reason)
	}
}

func TestResolveRepoParsesGitRemoteWithTrailingSlash(t *testing.T) {
	r := Router{MinConfidence: 0.75}

	res := r.Resolve(RouteInput{
		GitRemote: "https://github.com/gh-xj/agentcli-go/",
	})

	if res.Repo != "gh-xj/agentcli-go" {
		t.Fatalf("expected inferred repo, got %q", res.Repo)
	}
	if res.Pending {
		t.Fatalf("expected strong remote signal to be non-pending")
	}
	if res.Reason != "git_remote" {
		t.Fatalf("expected git_remote reason, got %q", res.Reason)
	}
}

func TestResolveRepoParsesSSHRemoteWithPort(t *testing.T) {
	r := Router{MinConfidence: 0.75}

	res := r.Resolve(RouteInput{
		GitRemote: "ssh://git@github.com:2222/gh-xj/agentcli-go.git",
	})

	if res.Repo != "gh-xj/agentcli-go" {
		t.Fatalf("expected inferred repo, got %q", res.Repo)
	}
	if res.Pending {
		t.Fatalf("expected strong remote signal to be non-pending")
	}
	if res.Reason != "git_remote" {
		t.Fatalf("expected git_remote reason, got %q", res.Reason)
	}
}
