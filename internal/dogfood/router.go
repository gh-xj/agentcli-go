package dogfood

import (
	"path/filepath"
	"regexp"
	"strings"
)

type RouteInput struct {
	OverrideRepo string
	CWD          string
	GitRemote    string
}

type RouteResult struct {
	Repo       string
	Confidence float64
	Reason     string
	Pending    bool
}

var githubRemotePattern = regexp.MustCompile(`(?i)github\.com[:/](?:\d+/)?([^/\s]+)/([^/\s]+?)(?:\.git)?/?$`)

func (r Router) Resolve(in RouteInput) RouteResult {
	override := strings.TrimSpace(in.OverrideRepo)
	if override != "" {
		return RouteResult{
			Repo:       override,
			Confidence: 1.0,
			Reason:     "manual_override",
		}
	}

	repo, confidence, reason := inferRepo(in)
	result := RouteResult{
		Repo:       repo,
		Confidence: confidence,
		Reason:     reason,
	}
	result.Pending = result.Confidence < r.minConfidence()

	return result
}

func inferRepo(in RouteInput) (string, float64, string) {
	if repo, ok := inferFromGitRemote(in.GitRemote); ok {
		return repo, 0.9, "git_remote"
	}
	if repo, ok := inferFromCWD(in.CWD); ok {
		return repo, 0.4, "cwd_guess"
	}
	return "", 0.0, "no_signal"
}

func inferFromGitRemote(remote string) (string, bool) {
	remote = strings.TrimSpace(remote)
	m := githubRemotePattern.FindStringSubmatch(remote)
	if len(m) != 3 {
		return "", false
	}

	owner := strings.TrimSpace(m[1])
	repo := strings.TrimSpace(m[2])
	if owner == "" || repo == "" {
		return "", false
	}

	return owner + "/" + repo, true
}

func inferFromCWD(cwd string) (string, bool) {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return "", false
	}

	base := filepath.Base(filepath.Clean(cwd))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", false
	}

	return base, true
}
