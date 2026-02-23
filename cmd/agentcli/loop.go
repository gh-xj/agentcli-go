package main

import (
	"encoding/json"
	"fmt"
	"os"

	agentcli "github.com/gh-xj/agentcli-go"
	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
	"github.com/gh-xj/agentcli-go/internal/loopapi"
)

func runLoop(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: agentcli loop [run|judge|autofix|all] [--threshold score] [--max-iterations n] [--repo-root path] [--branch name] [--api url]")
		return agentcli.ExitUsage
	}

	action := args[0]
	repoRoot, threshold, maxIterations, branch, apiURL, err := parseLoopFlags(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitUsage
	}

	var result harnessloop.RunResult
	if apiURL != "" {
		result, err = loopapi.Run(apiURL, loopapi.RunRequest{
			Action:        action,
			RepoRoot:      repoRoot,
			Threshold:     threshold,
			MaxIterations: maxIterations,
			Branch:        branch,
		})
	} else {
		cfg := harnessloop.Config{
			RepoRoot:      repoRoot,
			Threshold:     threshold,
			MaxIterations: maxIterations,
			Branch:        branch,
		}

		switch action {
		case "run", "judge":
			cfg.AutoFix = false
			cfg.AutoCommit = false
		case "autofix", "all":
			cfg.AutoFix = true
			cfg.AutoCommit = true
		default:
			fmt.Fprintf(os.Stderr, "unknown loop action: %s\n", action)
			return agentcli.ExitUsage
		}
		result, err = harnessloop.RunLoop(cfg)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitFailure
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Fprintln(os.Stdout, string(out))
	if result.Judge.Pass {
		return agentcli.ExitSuccess
	}
	return agentcli.ExitFailure
}

func parseLoopFlags(args []string) (string, float64, int, string, string, error) {
	repoRoot := "."
	threshold := 9.0
	maxIterations := 3
	branch := "autofix/onboarding-loop"
	apiURL := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-root":
			if i+1 >= len(args) {
				return "", 0, 0, "", "", fmt.Errorf("--repo-root requires a value")
			}
			repoRoot = args[i+1]
			i++
		case "--threshold":
			if i+1 >= len(args) {
				return "", 0, 0, "", "", fmt.Errorf("--threshold requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &threshold); err != nil {
				return "", 0, 0, "", "", fmt.Errorf("invalid --threshold value")
			}
			i++
		case "--max-iterations":
			if i+1 >= len(args) {
				return "", 0, 0, "", "", fmt.Errorf("--max-iterations requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &maxIterations); err != nil {
				return "", 0, 0, "", "", fmt.Errorf("invalid --max-iterations value")
			}
			i++
		case "--branch":
			if i+1 >= len(args) {
				return "", 0, 0, "", "", fmt.Errorf("--branch requires a value")
			}
			branch = args[i+1]
			i++
		case "--api":
			if i+1 >= len(args) {
				return "", 0, 0, "", "", fmt.Errorf("--api requires a value")
			}
			apiURL = args[i+1]
			i++
		default:
			return "", 0, 0, "", "", fmt.Errorf("unexpected argument: %s", args[i])
		}
	}
	return repoRoot, threshold, maxIterations, branch, apiURL, nil
}
