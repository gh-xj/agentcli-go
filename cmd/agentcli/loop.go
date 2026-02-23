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
		fmt.Fprintln(os.Stderr, "usage: agentcli loop [run|judge|autofix|doctor|lab]")
		return agentcli.ExitUsage
	}

	if args[0] == "lab" {
		return runLoopLab(args[1:])
	}

	action := args[0]
	if action == "doctor" {
		opts, err := parseLoopFlags(args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return agentcli.ExitUsage
		}
		report := harnessloop.LoopDoctor(opts.RepoRoot)
		if opts.Markdown {
			fmt.Fprintln(os.Stdout, harnessloop.RenderDoctorMarkdown(report))
		} else {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Fprintln(os.Stdout, string(out))
		}
		if report.LeanReady {
			return agentcli.ExitSuccess
		}
		return agentcli.ExitFailure
	}

	if action != "run" && action != "judge" && action != "autofix" {
		fmt.Fprintf(os.Stderr, "unknown loop action: %s\n", action)
		fmt.Fprintln(os.Stderr, "use 'agentcli loop lab' for advanced actions")
		return agentcli.ExitUsage
	}

	opts, err := parseLoopFlags(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitUsage
	}

	cfg := harnessloop.Config{
		RepoRoot:         opts.RepoRoot,
		Threshold:        opts.Threshold,
		MaxIterations:    opts.MaxIterations,
		Branch:           opts.Branch,
		Mode:             "committee",
		RoleConfigPath:   "",
		Budget:           1,
		VerboseArtifacts: false,
	}
	if action == "autofix" {
		cfg.AutoFix = true
		cfg.AutoCommit = true
	}

	result, err := runLoopWithOptionalAPI(opts.APIURL, action, cfg)
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

func runLoopLab(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: agentcli loop lab [compare|replay|run|judge|autofix] ...")
		return agentcli.ExitUsage
	}
	action := args[0]
	opts, err := parseLoopLabFlags(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitUsage
	}

	switch action {
	case "compare":
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
		if path, err := harnessloop.WriteCompareOutput(opts.RepoRoot, report, opts.Format, opts.Out); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return agentcli.ExitFailure
		} else if path != "" {
			fmt.Fprintf(os.Stdout, "compare report written: %s\n", path)
			return agentcli.ExitSuccess
		}
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(os.Stdout, string(out))
		return agentcli.ExitSuccess
	case "replay":
		if opts.APIURL != "" {
			fmt.Fprintln(os.Stderr, "replay action is local-only; remove --api")
			return agentcli.ExitUsage
		}
		if opts.RunID == "" || opts.Iteration <= 0 {
			fmt.Fprintln(os.Stderr, "replay action requires --run-id and --iter")
			return agentcli.ExitUsage
		}
		report, err := harnessloop.ReplayIteration(opts.RepoRoot, opts.RunID, opts.Iteration, opts.Threshold)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return agentcli.ExitFailure
		}
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(os.Stdout, string(out))
		if report.ReplayJudge.Pass {
			return agentcli.ExitSuccess
		}
		return agentcli.ExitFailure
	case "run", "judge", "autofix":
		cfg := harnessloop.Config{
			RepoRoot:         opts.RepoRoot,
			Threshold:        opts.Threshold,
			MaxIterations:    opts.MaxIterations,
			Branch:           opts.Branch,
			Mode:             opts.Mode,
			RoleConfigPath:   opts.RoleConfig,
			Seed:             opts.Seed,
			Budget:           opts.Budget,
			VerboseArtifacts: opts.VerboseArtifacts,
		}
		if action == "autofix" {
			cfg.AutoFix = true
			cfg.AutoCommit = true
		}
		result, err := runLoopWithOptionalAPI(opts.APIURL, action, cfg)
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
	default:
		fmt.Fprintf(os.Stderr, "unknown lab action: %s\n", action)
		return agentcli.ExitUsage
	}
}

func runLoopWithOptionalAPI(apiURL, action string, cfg harnessloop.Config) (harnessloop.RunResult, error) {
	if apiURL != "" {
		return loopapi.Run(apiURL, loopapi.RunRequest{
			Action:        action,
			RepoRoot:      cfg.RepoRoot,
			Threshold:     cfg.Threshold,
			MaxIterations: cfg.MaxIterations,
			Branch:        cfg.Branch,
			Mode:          cfg.Mode,
			RoleConfig:    cfg.RoleConfigPath,
			Seed:          cfg.Seed,
			Budget:        cfg.Budget,
		})
	}
	return harnessloop.RunLoop(cfg)
}

type loopFlags struct {
	RepoRoot      string
	Threshold     float64
	MaxIterations int
	Branch        string
	APIURL        string
	Markdown      bool
}

func parseLoopFlags(args []string) (loopFlags, error) {
	opts := loopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
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
		case "--md":
			opts.Markdown = true
		default:
			return loopFlags{}, fmt.Errorf("unexpected argument: %s", args[i])
		}
	}
	return opts, nil
}

type loopLabFlags struct {
	RepoRoot         string
	Threshold        float64
	MaxIterations    int
	Branch           string
	APIURL           string
	Mode             string
	RoleConfig       string
	Seed             int64
	Budget           int
	RunA             string
	RunB             string
	RunID            string
	Iteration        int
	Format           string
	Out              string
	VerboseArtifacts bool
}

func parseLoopLabFlags(args []string) (loopLabFlags, error) {
	opts := loopLabFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
		Mode:          "committee",
		Budget:        1,
		Format:        "json",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-root":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--repo-root requires a value")
			}
			opts.RepoRoot = args[i+1]
			i++
		case "--threshold":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--threshold requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &opts.Threshold); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --threshold value")
			}
			i++
		case "--max-iterations":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--max-iterations requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.MaxIterations); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --max-iterations value")
			}
			i++
		case "--branch":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--branch requires a value")
			}
			opts.Branch = args[i+1]
			i++
		case "--api":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--api requires a value")
			}
			opts.APIURL = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--mode requires a value")
			}
			opts.Mode = args[i+1]
			i++
		case "--role-config":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = args[i+1]
			i++
		case "--seed":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--seed requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.Seed); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --seed value")
			}
			i++
		case "--budget":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--budget requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.Budget); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --budget value")
			}
			i++
		case "--run-a":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--run-a requires a value")
			}
			opts.RunA = args[i+1]
			i++
		case "--run-b":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--run-b requires a value")
			}
			opts.RunB = args[i+1]
			i++
		case "--run-id":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--run-id requires a value")
			}
			opts.RunID = args[i+1]
			i++
		case "--iter":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--iter requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.Iteration); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --iter value")
			}
			i++
		case "--format":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--format requires a value")
			}
			opts.Format = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				return loopLabFlags{}, fmt.Errorf("--out requires a value")
			}
			opts.Out = args[i+1]
			i++
		case "--verbose-artifacts":
			opts.VerboseArtifacts = true
		default:
			return loopLabFlags{}, fmt.Errorf("unexpected argument: %s", args[i])
		}
	}

	if opts.Mode != "classic" && opts.Mode != "committee" {
		return loopLabFlags{}, fmt.Errorf("invalid --mode value: %s", opts.Mode)
	}
	return opts, nil
}
