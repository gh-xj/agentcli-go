package dogfood

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	PublishActionCreated   = "created"
	PublishActionCommented = "commented"
)

type CommandRunner interface {
	Run(name string, args ...string) (stdout string, err error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			return "", fmt.Errorf("run command %q: %w", name, err)
		}
		return "", fmt.Errorf("run command %q: %w: %s", name, err, trimmed)
	}
	return string(out), nil
}

type PublishInput struct {
	Repo             string
	Title            string
	Body             string
	ExistingIssueURL string
}

type Publisher struct {
	Runner CommandRunner
}

func (p Publisher) Publish(in PublishInput) (issueURL, action string, err error) {
	runner := p.runner()
	existingURL := strings.TrimSpace(in.ExistingIssueURL)
	if existingURL != "" {
		if _, err := runner.Run("gh", "issue", "comment", existingURL, "--body", in.Body); err != nil {
			return "", "", fmt.Errorf("comment issue %q: %w", existingURL, err)
		}
		return existingURL, PublishActionCommented, nil
	}

	repo := strings.TrimSpace(in.Repo)
	if repo == "" {
		return "", "", errors.New("repo is required when creating an issue")
	}

	out, err := runner.Run("gh", "issue", "create", "--repo", repo, "--title", strings.TrimSpace(in.Title), "--body", in.Body)
	if err != nil {
		return "", "", fmt.Errorf("create issue in %q: %w", repo, err)
	}

	return strings.TrimSpace(out), PublishActionCreated, nil
}

func (p Publisher) runner() CommandRunner {
	if p.Runner == nil {
		return ExecCommandRunner{}
	}
	return p.Runner
}
