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
		fmt.Fprintln(os.Stderr, "usage: agentcli loop [run|judge|autofix|all|compare] [--threshold score] [--max-iterations n] [--repo-root path] [--branch name] [--mode classic|committee] [--role-config file] [--seed n] [--budget n] [--run-a ref] [--run-b ref] [--api url]")
		return agentcli.ExitUsage
	}

	action := args[0]
	opts, err := parseLoopFlags(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitUsage
	}

	var result harnessloop.RunResult
	if action == "compare" {
		if opts.APIURL != "" {
			fmt.Fprintln(os.Stderr, "compare action is local-only; remove --api")
			return agentcli.ExitUsage
		}
		if opts.RunA == "" || opts.RunB == "" {
			fmt.Fprintln(os.Stderr, "compare action requires --run-a and --run-b")
			return agentcli.ExitUsage
		}
		report, err := harnessloop.CompareRuns(opts.RepoRoot, opts.RunA, opts.RunB)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return agentcli.ExitFailure
		}
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(os.Stdout, string(out))
		return agentcli.ExitSuccess
	}
	if opts.APIURL != "" {
		result, err = loopapi.Run(opts.APIURL, loopapi.RunRequest{
			Action:        action,
			RepoRoot:      opts.RepoRoot,
			Threshold:     opts.Threshold,
			MaxIterations: opts.MaxIterations,
			Branch:        opts.Branch,
			Mode:          opts.Mode,
			RoleConfig:    opts.RoleConfig,
			Seed:          opts.Seed,
			Budget:        opts.Budget,
		})
	} else {
		cfg := harnessloop.Config{
			RepoRoot:       opts.RepoRoot,
			Threshold:      opts.Threshold,
			MaxIterations:  opts.MaxIterations,
			Branch:         opts.Branch,
			Mode:           opts.Mode,
			RoleConfigPath: opts.RoleConfig,
			Seed:           opts.Seed,
			Budget:         opts.Budget,
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

type loopFlags struct {
	RepoRoot      string
	Threshold     float64
	MaxIterations int
	Branch        string
	APIURL        string
	Mode          string
	RoleConfig    string
	Seed          int64
	Budget        int
	RunA          string
	RunB          string
}

func parseLoopFlags(args []string) (loopFlags, error) {
	opts := loopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
		Mode:          "classic",
		Budget:        1,
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-root":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--repo-root requires a value")
			}
			opts.RepoRoot = args[i+1]
			i++
		case "--threshold":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--threshold requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &opts.Threshold); err != nil {
				return loopFlags{}, fmt.Errorf("invalid --threshold value")
			}
			i++
		case "--max-iterations":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--max-iterations requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.MaxIterations); err != nil {
				return loopFlags{}, fmt.Errorf("invalid --max-iterations value")
			}
			i++
		case "--branch":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--branch requires a value")
			}
			opts.Branch = args[i+1]
			i++
		case "--api":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--api requires a value")
			}
			opts.APIURL = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--mode requires a value")
			}
			opts.Mode = args[i+1]
			i++
		case "--role-config":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = args[i+1]
			i++
		case "--seed":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--seed requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.Seed); err != nil {
				return loopFlags{}, fmt.Errorf("invalid --seed value")
			}
			i++
		case "--budget":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--budget requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.Budget); err != nil {
				return loopFlags{}, fmt.Errorf("invalid --budget value")
			}
			i++
		case "--run-a":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--run-a requires a value")
			}
			opts.RunA = args[i+1]
			i++
		case "--run-b":
			if i+1 >= len(args) {
				return loopFlags{}, fmt.Errorf("--run-b requires a value")
			}
			opts.RunB = args[i+1]
			i++
		default:
			return loopFlags{}, fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	if opts.Mode != "classic" && opts.Mode != "committee" {
		return loopFlags{}, fmt.Errorf("invalid --mode value: %s", opts.Mode)
	}
	return opts, nil
}
