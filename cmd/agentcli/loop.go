package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
	"github.com/gh-xj/agentcli-go/internal/loopapi"
	harness "github.com/gh-xj/agentcli-go/tools/harness"
)

type loopProfile struct {
	mode             string
	roleConfig       string
	maxIterations    int
	threshold        float64
	budget           int
	verboseArtifacts bool
}

type loopProfileJSON struct {
	Mode             string  `json:"mode"`
	RoleConfig       string  `json:"role_config"`
	MaxIterations    int     `json:"max_iterations"`
	Threshold        float64 `json:"threshold"`
	Budget           int     `json:"budget"`
	VerboseArtifacts bool    `json:"verbose_artifacts"`
}

const loopProfilesConfigFile = "configs/loop-profiles.json"

var loopProfiles = map[string]loopProfile{
	"quality": {
		mode:             "committee",
		roleConfig:       "configs/skill-quality.roles.json",
		maxIterations:    1,
		threshold:        9.0,
		budget:           1,
		verboseArtifacts: true,
	},
}

func getLoopProfiles(repoRoot string) (map[string]loopProfile, error) {
	profiles := make(map[string]loopProfile, len(loopProfiles)+1)
	for name, profile := range loopProfiles {
		profiles[name] = profile
	}

	path := filepath.Join(repoRoot, loopProfilesConfigFile)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return profiles, nil
		}
		return nil, fmt.Errorf("read loop profiles: %w", err)
	}

	var fileProfiles map[string]loopProfileJSON
	if err := json.Unmarshal(raw, &fileProfiles); err != nil {
		return nil, fmt.Errorf("parse loop profiles: %w", err)
	}
	for name, profile := range fileProfiles {
		profiles[name] = loopProfile{
			mode:             profile.Mode,
			roleConfig:       profile.RoleConfig,
			maxIterations:    profile.MaxIterations,
			threshold:        profile.Threshold,
			budget:           profile.Budget,
			verboseArtifacts: profile.VerboseArtifacts,
		}
	}
	return profiles, nil
}

func formatLoopProfile(name string, profile loopProfile) string {
	mode := profile.mode
	if mode == "" {
		mode = "(not set)"
	}
	roleConfig := profile.roleConfig
	if roleConfig == "" {
		roleConfig = "(not set)"
	}
	return fmt.Sprintf("%s: mode=%s threshold=%.1f max_iterations=%d budget=%d role_config=%s verbose_artifacts=%t",
		name, mode, profile.threshold, profile.maxIterations, profile.budget, roleConfig, profile.verboseArtifacts)
}

func parseLoopProfilesRepoRoot(args []string) (string, error) {
	repoRoot := "."
	for i := 0; i < len(args); i++ {
		if args[i] != "--repo-root" {
			continue
		}
		if i+1 >= len(args) {
			return "", fmt.Errorf("--repo-root requires a value")
		}
		repoRoot = args[i+1]
		i++
	}
	return repoRoot, nil
}

type loopRuntimeFlags struct {
	Format      string
	SummaryPath string
	NoColor     bool
	DryRun      bool
	Explain     bool
}

type loopRegressionFlags struct {
	Profile       string
	BaselinePath  string
	WriteBaseline bool
}

type loopRegressionReport struct {
	SchemaVersion   string                        `json:"schema_version"`
	Profile         string                        `json:"profile"`
	BaselinePath    string                        `json:"baseline_path"`
	BaselineWritten bool                          `json:"baseline_written,omitempty"`
	RunID           string                        `json:"run_id,omitempty"`
	Pass            bool                          `json:"pass"`
	DriftCount      int                           `json:"drift_count"`
	Drifts          []harnessloop.RegressionDrift `json:"drifts,omitempty"`
}

func runLoop(args []string) int {
	runtime := loopRuntimeFlags{Format: "text"}
	if len(args) == 0 {
		return emitLoopFailureSummary(
			"loop",
			runtime,
			harness.NewFailure(
				harness.CodeUsage,
				"usage: agentcli loop [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [--format text|json|ndjson] [--summary path] [--no-color] [--dry-run] [--explain]",
				"",
				false,
			),
		)
	}

	parsedRuntime, remaining, err := parseLoopRuntimeFlags(args)
	runtime = parsedRuntime
	if err != nil {
		return emitLoopFailureSummary("loop", runtime, err)
	}
	if len(remaining) == 0 {
		return emitLoopFailureSummary(
			"loop",
			runtime,
			harness.NewFailure(
				harness.CodeUsage,
				"usage: agentcli loop [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab]",
				"",
				false,
			),
		)
	}

	action := remaining[0]
	actionArgs := remaining[1:]
	summary, execErr := harness.Run(harness.CommandInput{
		Name:        "loop " + action,
		SummaryPath: runtime.SummaryPath,
		DryRun:      runtime.DryRun,
		Explain:     runtime.Explain,
		Execute: func(ctx harness.Context) (harness.CommandOutcome, error) {
			return executeLoopAction(action, actionArgs, ctx)
		},
	})

	rendered, renderErr := harness.RenderSummary(summary, runtime.Format, runtime.NoColor)
	if renderErr != nil {
		fmt.Fprintln(os.Stderr, renderErr.Error())
		return harness.ExitCodeFor(renderErr)
	}
	fmt.Fprint(os.Stdout, rendered)
	return harness.ExitCodeFor(execErr)
}

func executeLoopAction(action string, args []string, _ harness.Context) (harness.CommandOutcome, error) {
	switch action {
	case "capabilities":
		return harness.CommandOutcome{
			Data: harness.DefaultCapabilities(),
		}, nil
	case "doctor":
		return runLoopDoctorCommand(args)
	case "profiles":
		return runLoopProfilesCommand(args)
	case "quality":
		return runLoopProfileCommand("quality", args)
	case "regression":
		return runLoopRegressionCommand(args)
	case "profile":
		if len(args) == 0 {
			return harness.CommandOutcome{}, harness.NewFailure(
				harness.CodeUsage,
				"usage: agentcli loop profile <name> [--repo-root path] [--threshold score] [--max-iterations n] [--branch name] [--api url] [--role-config path] [--verbose-artifacts|--no-verbose-artifacts]",
				"",
				false,
			)
		}
		return runLoopProfileCommand(args[0], args[1:])
	case "lab":
		return runLoopLabCommand(args)
	case "run", "judge", "autofix":
		return runLoopClassicCommand(action, args)
	default:
		if out, err, ok := runLoopNamedProfileCommand(action, args); ok {
			return out, err
		}
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown loop action: %s", action),
			"use 'agentcli loop capabilities --format json' to discover commands",
			false,
		)
	}
}

func runLoopNamedProfileCommand(name string, args []string) (harness.CommandOutcome, error, bool) {
	repoRoot, err := parseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err), true
	}
	profiles, err := getLoopProfiles(repoRoot)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "read loop profiles", "", false, err), true
	}
	if _, ok := profiles[name]; !ok {
		return harness.CommandOutcome{}, nil, false
	}
	out, runErr := runLoopProfileCommand(name, args)
	return out, runErr, true
}

func runLoopDoctorCommand(args []string) (harness.CommandOutcome, error) {
	opts, err := parseLoopFlags(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	report := harnessloop.LoopDoctor(opts.RepoRoot)
	data := any(report)
	if opts.Markdown {
		data = map[string]any{
			"report":   report,
			"markdown": harnessloop.RenderDoctorMarkdown(report),
		}
	}

	outcome := harness.CommandOutcome{
		Checks: []harness.CheckResult{
			{Name: "lean_ready", Status: boolStatus(report.LeanReady)},
			{Name: "lab_features_ready", Status: boolStatus(report.LabFeaturesReady)},
		},
		Data: data,
	}
	if !report.LeanReady {
		outcome.Failures = append(outcome.Failures, harness.Failure{
			Code:      string(harness.CodeContractValidation),
			Message:   "loop doctor found readiness issues",
			Hint:      "fix findings before running loop quality",
			Retryable: false,
		})
	}
	return outcome, nil
}

func runLoopProfilesCommand(args []string) (harness.CommandOutcome, error) {
	repoRoot, err := parseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	profiles, err := getLoopProfiles(repoRoot)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "read loop profiles", "", false, err)
	}

	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, formatLoopProfile(name, profiles[name]))
	}

	return harness.CommandOutcome{
		Checks: []harness.CheckResult{
			{
				Name:    "profiles_loaded",
				Status:  "ok",
				Details: fmt.Sprintf("%d profile(s)", len(names)),
			},
		},
		Data: map[string]any{
			"repo_root": repoRoot,
			"profiles":  lines,
		},
	}, nil
}

func runLoopClassicCommand(action string, args []string) (harness.CommandOutcome, error) {
	if action == "judge" {
		action = "run"
	}

	opts, err := parseLoopFlags(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
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
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop run failed", "", false, err)
	}
	return outcomeFromRunResult(result), nil
}

func runLoopProfileCommand(name string, args []string) (harness.CommandOutcome, error) {
	repoRoot, err := parseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	profiles, err := getLoopProfiles(repoRoot)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "read loop profiles", "", false, err)
	}

	profile, ok := profiles[name]
	if !ok {
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown loop profile: %s", name),
			"use 'agentcli loop profiles --format json' to inspect profiles",
			false,
		)
	}

	opts, err := parseLoopQualityFlags(profile, args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	roleConfig := profile.roleConfig
	if opts.RoleConfig != "" {
		roleConfig = opts.RoleConfig
	}
	roleConfigPath := roleConfig
	if opts.APIURL == "" {
		roleConfigPath = resolveRoleConfigPath(opts.RepoRoot, roleConfig)
	}
	verboseArtifacts, err := resolveVerboseArtifacts(profile.verboseArtifacts, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	cfg := harnessloop.Config{
		RepoRoot:         opts.RepoRoot,
		Threshold:        opts.Threshold,
		MaxIterations:    opts.MaxIterations,
		Branch:           opts.Branch,
		Mode:             profile.mode,
		RoleConfigPath:   roleConfigPath,
		Budget:           profile.budget,
		VerboseArtifacts: verboseArtifacts,
	}

	result, err := runLoopWithOptionalAPI(opts.APIURL, "judge", cfg)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop profile run failed", "", false, err)
	}

	outcome := outcomeFromRunResult(result)
	outcome.Checks = append([]harness.CheckResult{{
		Name:    "profile",
		Status:  "ok",
		Details: name,
	}}, outcome.Checks...)
	return outcome, nil
}

func runLoopRegressionCommand(args []string) (harness.CommandOutcome, error) {
	regressionFlags, profileArgs, err := parseLoopRegressionFlags(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	repoRoot, err := parseLoopProfilesRepoRoot(profileArgs)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	profiles, err := getLoopProfiles(repoRoot)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "read loop profiles", "", false, err)
	}
	profile, ok := profiles[regressionFlags.Profile]
	if !ok {
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown loop profile: %s", regressionFlags.Profile),
			"use 'agentcli loop profiles --format json' to inspect profiles",
			false,
		)
	}

	opts, err := parseLoopQualityFlags(profile, profileArgs)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	roleConfig := profile.roleConfig
	if opts.RoleConfig != "" {
		roleConfig = opts.RoleConfig
	}
	roleConfigPath := roleConfig
	if opts.APIURL == "" {
		roleConfigPath = resolveRoleConfigPath(opts.RepoRoot, roleConfig)
	}
	verboseArtifacts, err := resolveVerboseArtifacts(profile.verboseArtifacts, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	cfg := harnessloop.Config{
		RepoRoot:         opts.RepoRoot,
		Threshold:        opts.Threshold,
		MaxIterations:    opts.MaxIterations,
		Branch:           opts.Branch,
		Mode:             profile.mode,
		RoleConfigPath:   roleConfigPath,
		Budget:           profile.budget,
		VerboseArtifacts: verboseArtifacts,
	}

	result, err := runLoopWithOptionalAPI(opts.APIURL, "judge", cfg)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop regression run failed", "", false, err)
	}

	snapshot := harnessloop.BuildBehaviorSnapshot(result)
	baselinePath := resolveLoopRegressionBaselinePath(opts.RepoRoot, regressionFlags.Profile, regressionFlags.BaselinePath)
	if regressionFlags.WriteBaseline {
		baseline := harnessloop.RegressionBaseline{
			SchemaVersion: "v1",
			Kind:          "loop_behavior",
			Profile:       regressionFlags.Profile,
			GeneratedAt:   time.Now().UTC(),
			Snapshot:      snapshot,
		}
		if err := harnessloop.WriteRegressionBaseline(baselinePath, baseline); err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "write regression baseline", "", false, err)
		}
		return harness.CommandOutcome{
			Checks: []harness.CheckResult{
				{Name: "baseline_written", Status: "ok", Details: baselinePath},
			},
			Artifacts: []harness.Artifact{
				{Name: "behavior-baseline", Kind: "json", Path: baselinePath},
			},
			Data: loopRegressionReport{
				SchemaVersion:   "v1",
				Profile:         regressionFlags.Profile,
				BaselinePath:    baselinePath,
				BaselineWritten: true,
				RunID:           result.RunID,
				Pass:            true,
				DriftCount:      0,
			},
		}, nil
	}

	baseline, err := harnessloop.ReadRegressionBaseline(baselinePath)
	if err != nil {
		return harness.CommandOutcome{
				Artifacts: []harness.Artifact{
					{Name: "behavior-baseline", Kind: "json", Path: baselinePath},
				},
				Data: loopRegressionReport{
					SchemaVersion: "v1",
					Profile:       regressionFlags.Profile,
					BaselinePath:  baselinePath,
					Pass:          false,
				},
			}, harness.WrapFailure(
				harness.CodeContractValidation,
				"regression baseline missing or invalid",
				fmt.Sprintf("create baseline with: agentcli loop regression --repo-root %s --profile %s --write-baseline", opts.RepoRoot, regressionFlags.Profile),
				false,
				err,
			)
	}

	drifts := harnessloop.CompareBehaviorSnapshot(baseline.Snapshot, snapshot)
	report := loopRegressionReport{
		SchemaVersion: "v1",
		Profile:       regressionFlags.Profile,
		BaselinePath:  baselinePath,
		RunID:         result.RunID,
		Pass:          len(drifts) == 0,
		DriftCount:    len(drifts),
		Drifts:        drifts,
	}
	outcome := harness.CommandOutcome{
		Checks: []harness.CheckResult{
			{
				Name:    "behavior_drift",
				Status:  boolStatus(len(drifts) == 0),
				Details: fmt.Sprintf("%d drift(s)", len(drifts)),
			},
		},
		Artifacts: []harness.Artifact{
			{Name: "behavior-baseline", Kind: "json", Path: baselinePath},
		},
		Data: report,
	}
	if !report.Pass {
		outcome.Failures = append(outcome.Failures, harness.Failure{
			Code:      string(harness.CodeContractValidation),
			Message:   "loop behavior drift detected",
			Hint:      "run with --write-baseline only after intentional behavior changes",
			Retryable: false,
		})
	}
	return outcome, nil
}

func runLoopLabCommand(args []string) (harness.CommandOutcome, error) {
	if len(args) == 0 {
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			"usage: agentcli loop lab [compare|replay|run|judge|autofix] ...",
			"",
			false,
		)
	}
	action := args[0]
	opts, err := parseLoopLabFlags(args[1:])
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	switch action {
	case "compare":
		if opts.APIURL != "" {
			return harness.CommandOutcome{}, harness.NewFailure(harness.CodeUsage, "compare action is local-only; remove --api", "", false)
		}
		if opts.RunA == "" || opts.RunB == "" {
			return harness.CommandOutcome{}, harness.NewFailure(harness.CodeUsage, "compare action requires --run-a and --run-b", "", false)
		}
		report, err := harnessloop.CompareRuns(opts.RepoRoot, opts.RunA, opts.RunB)
		if err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "compare runs failed", "", false, err)
		}
		outcome := harness.CommandOutcome{
			Data: report,
		}
		if path, err := harnessloop.WriteCompareOutput(opts.RepoRoot, report, opts.Format, opts.Out); err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "write compare report", "", false, err)
		} else if path != "" {
			outcome.Artifacts = append(outcome.Artifacts, harness.Artifact{
				Name: "compare-report",
				Kind: opts.Format,
				Path: path,
			})
		}
		return outcome, nil
	case "replay":
		if opts.APIURL != "" {
			return harness.CommandOutcome{}, harness.NewFailure(harness.CodeUsage, "replay action is local-only; remove --api", "", false)
		}
		if opts.RunID == "" || opts.Iteration <= 0 {
			return harness.CommandOutcome{}, harness.NewFailure(harness.CodeUsage, "replay action requires --run-id and --iter", "", false)
		}
		report, err := harnessloop.ReplayIteration(opts.RepoRoot, opts.RunID, opts.Iteration, opts.Threshold)
		if err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "replay failed", "", false, err)
		}
		outcome := harness.CommandOutcome{
			Checks: []harness.CheckResult{
				{Name: "replay_pass", Status: boolStatus(report.ReplayJudge.Pass)},
			},
			Data: report,
		}
		if !report.ReplayJudge.Pass {
			outcome.Failures = append(outcome.Failures, harness.Failure{
				Code:      string(harness.CodeExecution),
				Message:   "replay judge failed",
				Retryable: false,
			})
		}
		return outcome, nil
	case "run", "judge", "autofix":
		roleConfigPath := opts.RoleConfig
		if opts.APIURL == "" {
			roleConfigPath = resolveRoleConfigPath(opts.RepoRoot, opts.RoleConfig)
		}
		verboseArtifacts, err := resolveVerboseArtifacts(false, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
		if err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
		}
		cfg := harnessloop.Config{
			RepoRoot:         opts.RepoRoot,
			Threshold:        opts.Threshold,
			MaxIterations:    opts.MaxIterations,
			Branch:           opts.Branch,
			Mode:             opts.Mode,
			RoleConfigPath:   roleConfigPath,
			Seed:             opts.Seed,
			Budget:           opts.Budget,
			VerboseArtifacts: verboseArtifacts,
		}
		if action == "autofix" {
			cfg.AutoFix = true
			cfg.AutoCommit = true
		}
		result, err := runLoopWithOptionalAPI(opts.APIURL, action, cfg)
		if err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop lab run failed", "", false, err)
		}
		return outcomeFromRunResult(result), nil
	default:
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown lab action: %s", action),
			"use 'agentcli loop capabilities --format json' to discover lab actions",
			false,
		)
	}
}

func runLoopWithOptionalAPI(apiURL, action string, cfg harnessloop.Config) (harnessloop.RunResult, error) {
	if apiURL != "" {
		return loopapi.Run(apiURL, loopapi.RunRequest{
			Action:           action,
			RepoRoot:         cfg.RepoRoot,
			Threshold:        cfg.Threshold,
			MaxIterations:    cfg.MaxIterations,
			Branch:           cfg.Branch,
			Mode:             cfg.Mode,
			RoleConfig:       cfg.RoleConfigPath,
			VerboseArtifacts: cfg.VerboseArtifacts,
			Seed:             cfg.Seed,
			Budget:           cfg.Budget,
		})
	}
	return harnessloop.RunLoop(cfg)
}

func outcomeFromRunResult(result harnessloop.RunResult) harness.CommandOutcome {
	outcome := harness.CommandOutcome{
		Checks: []harness.CheckResult{
			{
				Name:    "scenario_ok",
				Status:  boolStatus(result.Scenario.OK),
				Details: result.Scenario.Name,
			},
			{
				Name:    "judge_pass",
				Status:  boolStatus(result.Judge.Pass),
				Details: fmt.Sprintf("score=%.2f threshold=%.2f", result.Judge.Score, result.Judge.Threshold),
			},
		},
		Data: result,
	}
	if result.RunID != "" {
		outcome.Artifacts = append(outcome.Artifacts, harness.Artifact{
			Name: "run-result",
			Kind: "json",
			Path: filepath.Join(".docs", "onboarding-loop", "runs", result.RunID, "run-result.json"),
		})
	}
	if !result.Judge.Pass {
		outcome.Failures = append(outcome.Failures, harness.Failure{
			Code:      string(harness.CodeExecution),
			Message:   "loop judge failed threshold",
			Hint:      "review findings and rerun with adjusted strategy or threshold",
			Retryable: false,
		})
	}
	return outcome
}

func boolStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "fail"
}

func parseLoopRuntimeFlags(args []string) (loopRuntimeFlags, []string, error) {
	flags := loopRuntimeFlags{
		Format: "text",
	}
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--format":
			if i+1 >= len(args) {
				return flags, nil, harness.NewFailure(harness.CodeUsage, "--format requires a value", "", false)
			}
			flags.Format = args[i+1]
			i++
		case "--summary":
			if i+1 >= len(args) {
				return flags, nil, harness.NewFailure(harness.CodeUsage, "--summary requires a value", "", false)
			}
			flags.SummaryPath = args[i+1]
			i++
		case "--no-color":
			flags.NoColor = true
		case "--dry-run":
			flags.DryRun = true
		case "--explain":
			flags.Explain = true
		default:
			remaining = append(remaining, args[i])
		}
	}
	switch flags.Format {
	case "text", "json", "ndjson":
	default:
		return flags, nil, harness.NewFailure(harness.CodeUsage, "invalid --format value", "use text|json|ndjson", false)
	}
	return flags, remaining, nil
}

func emitLoopFailureSummary(command string, runtime loopRuntimeFlags, err error) int {
	now := time.Now().UTC()
	summary := harness.CommandSummary{
		SchemaVersion: harness.SummarySchemaVersion,
		Command:       command,
		Status:        string(harness.StatusFail),
		StartedAt:     now,
		FinishedAt:    now,
		DurationMs:    0,
		Failures: []harness.Failure{
			harness.FailureFromError(err),
		},
	}
	format := runtime.Format
	if format == "" {
		format = "text"
	}
	rendered, renderErr := harness.RenderSummary(summary, format, runtime.NoColor)
	if renderErr != nil {
		fmt.Fprintln(os.Stderr, renderErr.Error())
		return harness.ExitCodeFor(renderErr)
	}
	fmt.Fprint(os.Stdout, rendered)
	return harness.ExitCodeFor(err)
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
	opts, remaining, err := parseLoopBaseFlags(args, loopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
	}, true)
	if err != nil {
		return loopFlags{}, err
	}
	if len(remaining) > 0 {
		return loopFlags{}, fmt.Errorf("unexpected argument: %s", remaining[0])
	}
	return opts, nil
}

func parseLoopBaseFlags(args []string, defaults loopFlags, allowMarkdown bool) (loopFlags, []string, error) {
	opts := defaults
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-root":
			if i+1 >= len(args) {
				return loopFlags{}, nil, fmt.Errorf("--repo-root requires a value")
			}
			opts.RepoRoot = args[i+1]
			i++
		case "--threshold":
			if i+1 >= len(args) {
				return loopFlags{}, nil, fmt.Errorf("--threshold requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &opts.Threshold); err != nil {
				return loopFlags{}, nil, fmt.Errorf("invalid --threshold value")
			}
			i++
		case "--max-iterations":
			if i+1 >= len(args) {
				return loopFlags{}, nil, fmt.Errorf("--max-iterations requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.MaxIterations); err != nil {
				return loopFlags{}, nil, fmt.Errorf("invalid --max-iterations value")
			}
			i++
		case "--branch":
			if i+1 >= len(args) {
				return loopFlags{}, nil, fmt.Errorf("--branch requires a value")
			}
			opts.Branch = args[i+1]
			i++
		case "--api":
			if i+1 >= len(args) {
				return loopFlags{}, nil, fmt.Errorf("--api requires a value")
			}
			opts.APIURL = args[i+1]
			i++
		case "--md":
			if allowMarkdown {
				opts.Markdown = true
			} else {
				remaining = append(remaining, args[i])
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return opts, remaining, nil
}

type loopProfileFlags struct {
	loopFlags
	RoleConfig         string
	VerboseArtifacts   bool
	NoVerboseArtifacts bool
}

func parseLoopQualityFlags(profile loopProfile, args []string) (loopProfileFlags, error) {
	base, remaining, err := parseLoopBaseFlags(args, loopFlags{
		RepoRoot:      ".",
		Threshold:     profile.threshold,
		MaxIterations: profile.maxIterations,
		Branch:        "autofix/onboarding-loop",
	}, true)
	if err != nil {
		return loopProfileFlags{}, err
	}

	opts := loopProfileFlags{
		loopFlags: base,
	}
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--role-config":
			if i+1 >= len(remaining) {
				return loopProfileFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = remaining[i+1]
			i++
		case "--verbose-artifacts":
			opts.VerboseArtifacts = true
		case "--no-verbose-artifacts":
			opts.NoVerboseArtifacts = true
		default:
			return loopProfileFlags{}, fmt.Errorf("unexpected argument: %s", remaining[i])
		}
	}
	if opts.VerboseArtifacts && opts.NoVerboseArtifacts {
		return loopProfileFlags{}, fmt.Errorf("cannot use --verbose-artifacts and --no-verbose-artifacts together")
	}
	return opts, nil
}

type loopLabFlags struct {
	loopFlags
	Mode               string
	RoleConfig         string
	Seed               int64
	Budget             int
	RunA               string
	RunB               string
	RunID              string
	Iteration          int
	Format             string
	Out                string
	VerboseArtifacts   bool
	NoVerboseArtifacts bool
}

func parseLoopLabFlags(args []string) (loopLabFlags, error) {
	base, remaining, err := parseLoopBaseFlags(args, loopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
	}, false)
	if err != nil {
		return loopLabFlags{}, err
	}

	opts := loopLabFlags{
		loopFlags: base,
		Mode:      "committee",
		Budget:    1,
		Format:    "json",
	}
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--mode":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--mode requires a value")
			}
			opts.Mode = remaining[i+1]
			i++
		case "--role-config":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = remaining[i+1]
			i++
		case "--seed":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--seed requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Seed); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --seed value")
			}
			i++
		case "--budget":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--budget requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Budget); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --budget value")
			}
			i++
		case "--run-a":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--run-a requires a value")
			}
			opts.RunA = remaining[i+1]
			i++
		case "--run-b":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--run-b requires a value")
			}
			opts.RunB = remaining[i+1]
			i++
		case "--run-id":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--run-id requires a value")
			}
			opts.RunID = remaining[i+1]
			i++
		case "--iter":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--iter requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Iteration); err != nil {
				return loopLabFlags{}, fmt.Errorf("invalid --iter value")
			}
			i++
		case "--format":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--format requires a value")
			}
			opts.Format = remaining[i+1]
			i++
		case "--out":
			if i+1 >= len(remaining) {
				return loopLabFlags{}, fmt.Errorf("--out requires a value")
			}
			opts.Out = remaining[i+1]
			i++
		case "--verbose-artifacts":
			opts.VerboseArtifacts = true
		case "--no-verbose-artifacts":
			opts.NoVerboseArtifacts = true
		default:
			return loopLabFlags{}, fmt.Errorf("unexpected argument: %s", remaining[i])
		}
	}
	if opts.VerboseArtifacts && opts.NoVerboseArtifacts {
		return loopLabFlags{}, fmt.Errorf("cannot use --verbose-artifacts and --no-verbose-artifacts together")
	}

	if opts.Mode != "classic" && opts.Mode != "committee" {
		return loopLabFlags{}, fmt.Errorf("invalid --mode value: %s", opts.Mode)
	}
	return opts, nil
}

func parseLoopRegressionFlags(args []string) (loopRegressionFlags, []string, error) {
	opts := loopRegressionFlags{
		Profile: "quality",
	}
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				return loopRegressionFlags{}, nil, fmt.Errorf("--profile requires a value")
			}
			opts.Profile = args[i+1]
			i++
		case "--baseline":
			if i+1 >= len(args) {
				return loopRegressionFlags{}, nil, fmt.Errorf("--baseline requires a value")
			}
			opts.BaselinePath = args[i+1]
			i++
		case "--write-baseline":
			opts.WriteBaseline = true
		default:
			remaining = append(remaining, args[i])
		}
	}
	if strings.TrimSpace(opts.Profile) == "" {
		return loopRegressionFlags{}, nil, fmt.Errorf("--profile requires a non-empty value")
	}
	return opts, remaining, nil
}

func resolveLoopRegressionBaselinePath(repoRoot, profileName, baselinePath string) string {
	if strings.TrimSpace(baselinePath) == "" {
		return filepath.Join(repoRoot, "testdata", "regression", fmt.Sprintf("loop-%s.behavior-baseline.json", profileName))
	}
	if filepath.IsAbs(baselinePath) {
		return baselinePath
	}
	return filepath.Join(repoRoot, baselinePath)
}

func resolveVerboseArtifacts(defaultValue, forceEnable, forceDisable bool) (bool, error) {
	if forceEnable && forceDisable {
		return false, fmt.Errorf("cannot use --verbose-artifacts and --no-verbose-artifacts together")
	}
	if forceEnable {
		return true, nil
	}
	if forceDisable {
		return false, nil
	}
	return defaultValue, nil
}

func resolveRoleConfigPath(repoRoot, roleConfig string) string {
	if roleConfig == "" {
		return ""
	}
	if filepath.IsAbs(roleConfig) {
		return roleConfig
	}
	return filepath.Join(repoRoot, roleConfig)
}
