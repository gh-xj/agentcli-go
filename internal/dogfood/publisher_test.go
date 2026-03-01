package dogfood

import (
	"reflect"
	"strings"
	"testing"
)

type fakeCommandRunner struct {
	out   string
	err   error
	calls []runnerCall
}

type runnerCall struct {
	name string
	args []string
}

func (f *fakeCommandRunner) Run(name string, args ...string) (string, error) {
	f.calls = append(f.calls, runnerCall{name: name, args: append([]string(nil), args...)})
	return f.out, f.err
}

func TestPublisherCreateIssueWhenNoExistingOpenRecord(t *testing.T) {
	runner := &fakeCommandRunner{out: "https://github.com/gh-xj/agentcli-go/issues/123\n"}
	pub := Publisher{Runner: runner}

	url, action, err := pub.Publish(PublishInput{
		Repo:  "gh-xj/agentcli-go",
		Title: "dogfood: runtime error",
		Body:  "details",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if action != PublishActionCreated {
		t.Fatalf("expected action %q, got %q", PublishActionCreated, action)
	}
	if !strings.Contains(url, "/issues/123") {
		t.Fatalf("expected created issue url, got %q", url)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly one runner call, got %d", len(runner.calls))
	}
	if runner.calls[0].name != "gh" {
		t.Fatalf("expected gh command, got %q", runner.calls[0].name)
	}
	wantArgs := []string{"issue", "create", "--repo", "gh-xj/agentcli-go", "--title", "dogfood: runtime error", "--body", "details"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", runner.calls[0].args, wantArgs)
	}
}

func TestPublisherCommentsExistingIssueWhenExistingIssueProvided(t *testing.T) {
	runner := &fakeCommandRunner{}
	pub := Publisher{Runner: runner}

	url, action, err := pub.Publish(PublishInput{
		Repo:             "gh-xj/agentcli-go",
		Title:            "dogfood: runtime error",
		Body:             "details",
		ExistingIssueURL: "https://github.com/gh-xj/agentcli-go/issues/123",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if action != PublishActionCommented {
		t.Fatalf("expected action %q, got %q", PublishActionCommented, action)
	}
	if url != "https://github.com/gh-xj/agentcli-go/issues/123" {
		t.Fatalf("expected existing issue url, got %q", url)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly one runner call, got %d", len(runner.calls))
	}
	wantArgs := []string{"issue", "comment", "https://github.com/gh-xj/agentcli-go/issues/123", "--body", "details"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", runner.calls[0].args, wantArgs)
	}
}
