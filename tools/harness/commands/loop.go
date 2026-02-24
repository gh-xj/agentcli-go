package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
	harness "github.com/gh-xj/agentcli-go/tools/harness"
)

const LoopProfilesConfigFile = "configs/loop-profiles.json"

// LoopProfiles contains built-in loop profiles.
var LoopProfiles = map[string]LoopProfile{
	"quality": {
		Mode:             "committee",
		RoleConfig:       "configs/skill-quality.roles.json",
		MaxIterations:    1,
		Threshold:        9.0,
		Budget:           1,
		VerboseArtifacts: true,
	},
}

type LoopProfile struct {
	Mode             string
	RoleConfig       string
	MaxIterations    int
	Threshold        float64
	Budget           int
	VerboseArtifacts bool
}

type loopProfileJSON struct {
	Mode             string  `json:"mode"`
	RoleConfig       string  `json:"role_config"`
	MaxIterations    int     `json:"max_iterations"`
	Threshold        float64 `json:"threshold"`
	Budget           int     `json:"budget"`
	VerboseArtifacts bool    `json:"verbose_artifacts"`
}

type LoopRuntimeFlags struct {
	Format      string
	SummaryPath string
	NoColor     bool
	DryRun      bool
	Explain     bool
}

type LoopRegressionFlags struct {
	Profile       string
	BaselinePath  string
	WriteBaseline bool
}

type LoopRegressionReport struct {
	SchemaVersion   string                        `json:"schema_version"`
	Profile         string                        `json:"profile"`
	BaselinePath    string                        `json:"baseline_path"`
	BaselineWritten bool                          `json:"baseline_written,omitempty"`
	RunID           string                        `json:"run_id,omitempty"`
	Pass            bool                          `json:"pass"`
	DriftCount      int                           `json:"drift_count"`
	Drifts          []harnessloop.RegressionDrift `json:"drifts,omitempty"`
}

type LoopFlags struct {
	RepoRoot      string
	Threshold     float64
	MaxIterations int
	Branch        string
	APIURL        string
	Markdown      bool
}

type LoopProfileFlags struct {
	LoopFlags
	RoleConfig         string
	VerboseArtifacts   bool
	NoVerboseArtifacts bool
}

type LoopLabFlags struct {
	LoopFlags
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

type RunLoopFunc func(apiURL, action string, cfg harnessloop.Config) (harnessloop.RunResult, error)

type LoopHandler struct {
	runLoop RunLoopFunc
}

func NewLoopHandler(runLoop RunLoopFunc) *LoopHandler {
	h := &LoopHandler{runLoop: runLoop}
	if h.runLoop == nil {
		h.runLoop = func(_ string, _ string, cfg harnessloop.Config) (harnessloop.RunResult, error) {
			return harnessloop.RunLoop(cfg)
		}
	}
	return h
}

func GetLoopProfiles(repoRoot string) (map[string]LoopProfile, error) {
	profiles := make(map[string]LoopProfile, len(LoopProfiles)+1)
	for name, profile := range LoopProfiles {
		profiles[name] = profile
	}

	path := filepath.Join(repoRoot, LoopProfilesConfigFile)
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
		profiles[name] = LoopProfile(profile)
	}
	return profiles, nil
}

func FormatLoopProfile(name string, profile LoopProfile) string {
	mode := profile.Mode
	if mode == "" {
		mode = "(not set)"
	}
	roleConfig := profile.RoleConfig
	if roleConfig == "" {
		roleConfig = "(not set)"
	}
	return fmt.Sprintf("%s: mode=%s threshold=%.1f max_iterations=%d budget=%d role_config=%s verbose_artifacts=%t",
		name, mode, profile.Threshold, profile.MaxIterations, profile.Budget, roleConfig, profile.VerboseArtifacts)
}

func ParseLoopProfilesRepoRoot(args []string) (string, error) {
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

func ParseLoopRuntimeFlags(args []string) (LoopRuntimeFlags, []string, error) {
	flags := LoopRuntimeFlags{Format: "text"}
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--format":
			if i+1 >= len(args) {
				return flags, nil, harness.NewFailure(harness.CodeUsage, "--format requires a value", "", false)
			}
			flags.Format = args[i+1]
			i += 2
		case "--summary":
			if i+1 >= len(args) {
				return flags, nil, harness.NewFailure(harness.CodeUsage, "--summary requires a value", "", false)
			}
			flags.SummaryPath = args[i+1]
			i += 2
		case "--no-color":
			flags.NoColor = true
			i++
		case "--dry-run":
			flags.DryRun = true
			i++
		case "--explain":
			flags.Explain = true
			i++
		default:
			return validateLoopRuntimeFlags(flags, args[i:])
		}
	}
	return validateLoopRuntimeFlags(flags, nil)
}

func validateLoopRuntimeFlags(flags LoopRuntimeFlags, remaining []string) (LoopRuntimeFlags, []string, error) {
	out := make([]string, len(remaining))
	copy(out, remaining)
	switch flags.Format {
	case "text", "json", "ndjson":
	default:
		return flags, nil, harness.NewFailure(harness.CodeUsage, "invalid --format value", "use text|json|ndjson", false)
	}
	return flags, out, nil
}

func ParseLoopFlags(args []string) (LoopFlags, error) {
	opts, remaining, err := parseLoopBaseFlags(args, LoopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
	}, true)
	if err != nil {
		return LoopFlags{}, err
	}
	if len(remaining) > 0 {
		return LoopFlags{}, UnexpectedLoopArgError(remaining[0])
	}
	return opts, nil
}

func UnexpectedLoopArgError(arg string) error {
	switch arg {
	case "--format", "--summary", "--no-color", "--dry-run", "--explain":
		return fmt.Errorf("unexpected argument: %s (%s is a global flag; place it before the action)", arg, arg)
	default:
		return fmt.Errorf("unexpected argument: %s", arg)
	}
}

func parseLoopBaseFlags(args []string, defaults LoopFlags, allowMarkdown bool) (LoopFlags, []string, error) {
	opts := defaults
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-root":
			if i+1 >= len(args) {
				return LoopFlags{}, nil, fmt.Errorf("--repo-root requires a value")
			}
			opts.RepoRoot = args[i+1]
			i++
		case "--threshold":
			if i+1 >= len(args) {
				return LoopFlags{}, nil, fmt.Errorf("--threshold requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%f", &opts.Threshold); err != nil {
				return LoopFlags{}, nil, fmt.Errorf("invalid --threshold value")
			}
			i++
		case "--max-iterations":
			if i+1 >= len(args) {
				return LoopFlags{}, nil, fmt.Errorf("--max-iterations requires a value")
			}
			if _, err := fmt.Sscanf(args[i+1], "%d", &opts.MaxIterations); err != nil {
				return LoopFlags{}, nil, fmt.Errorf("invalid --max-iterations value")
			}
			i++
		case "--branch":
			if i+1 >= len(args) {
				return LoopFlags{}, nil, fmt.Errorf("--branch requires a value")
			}
			opts.Branch = args[i+1]
			i++
		case "--api":
			if i+1 >= len(args) {
				return LoopFlags{}, nil, fmt.Errorf("--api requires a value")
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

func ParseLoopQualityFlags(profile LoopProfile, args []string) (LoopProfileFlags, error) {
	base, remaining, err := parseLoopBaseFlags(args, LoopFlags{
		RepoRoot:      ".",
		Threshold:     profile.Threshold,
		MaxIterations: profile.MaxIterations,
		Branch:        "autofix/onboarding-loop",
	}, false)
	if err != nil {
		return LoopProfileFlags{}, err
	}

	opts := LoopProfileFlags{LoopFlags: base}
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--role-config":
			if i+1 >= len(remaining) {
				return LoopProfileFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = remaining[i+1]
			i++
		case "--verbose-artifacts":
			opts.VerboseArtifacts = true
		case "--no-verbose-artifacts":
			opts.NoVerboseArtifacts = true
		default:
			return LoopProfileFlags{}, UnexpectedLoopArgError(remaining[i])
		}
	}
	if opts.VerboseArtifacts && opts.NoVerboseArtifacts {
		return LoopProfileFlags{}, fmt.Errorf("cannot use --verbose-artifacts and --no-verbose-artifacts together")
	}
	return opts, nil
}

func ParseLoopLabFlags(args []string) (LoopLabFlags, error) {
	base, remaining, err := parseLoopBaseFlags(args, LoopFlags{
		RepoRoot:      ".",
		Threshold:     9.0,
		MaxIterations: 3,
		Branch:        "autofix/onboarding-loop",
	}, false)
	if err != nil {
		return LoopLabFlags{}, err
	}

	opts := LoopLabFlags{
		LoopFlags: base,
		Mode:      "committee",
		Budget:    1,
		Format:    "json",
	}
	for i := 0; i < len(remaining); i++ {
		switch remaining[i] {
		case "--mode":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--mode requires a value")
			}
			opts.Mode = remaining[i+1]
			i++
		case "--role-config":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--role-config requires a value")
			}
			opts.RoleConfig = remaining[i+1]
			i++
		case "--seed":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--seed requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Seed); err != nil {
				return LoopLabFlags{}, fmt.Errorf("invalid --seed value")
			}
			i++
		case "--budget":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--budget requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Budget); err != nil {
				return LoopLabFlags{}, fmt.Errorf("invalid --budget value")
			}
			i++
		case "--run-a":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--run-a requires a value")
			}
			opts.RunA = remaining[i+1]
			i++
		case "--run-b":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--run-b requires a value")
			}
			opts.RunB = remaining[i+1]
			i++
		case "--run-id":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--run-id requires a value")
			}
			opts.RunID = remaining[i+1]
			i++
		case "--iter":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--iter requires a value")
			}
			if _, err := fmt.Sscanf(remaining[i+1], "%d", &opts.Iteration); err != nil {
				return LoopLabFlags{}, fmt.Errorf("invalid --iter value")
			}
			i++
		case "--format":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--format requires a value")
			}
			opts.Format = remaining[i+1]
			i++
		case "--out":
			if i+1 >= len(remaining) {
				return LoopLabFlags{}, fmt.Errorf("--out requires a value")
			}
			opts.Out = remaining[i+1]
			i++
		case "--verbose-artifacts":
			opts.VerboseArtifacts = true
		case "--no-verbose-artifacts":
			opts.NoVerboseArtifacts = true
		default:
			return LoopLabFlags{}, UnexpectedLoopArgError(remaining[i])
		}
	}
	if opts.VerboseArtifacts && opts.NoVerboseArtifacts {
		return LoopLabFlags{}, fmt.Errorf("cannot use --verbose-artifacts and --no-verbose-artifacts together")
	}
	if opts.Mode != "classic" && opts.Mode != "committee" {
		return LoopLabFlags{}, fmt.Errorf("invalid --mode value: %s", opts.Mode)
	}
	return opts, nil
}

func ParseLoopRegressionFlags(args []string) (LoopRegressionFlags, []string, error) {
	opts := LoopRegressionFlags{Profile: "quality"}
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				return LoopRegressionFlags{}, nil, fmt.Errorf("--profile requires a value")
			}
			opts.Profile = args[i+1]
			i++
		case "--baseline":
			if i+1 >= len(args) {
				return LoopRegressionFlags{}, nil, fmt.Errorf("--baseline requires a value")
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
		return LoopRegressionFlags{}, nil, fmt.Errorf("--profile requires a non-empty value")
	}
	return opts, remaining, nil
}

func ResolveLoopRegressionBaselinePath(repoRoot, profileName, baselinePath string) string {
	if strings.TrimSpace(baselinePath) == "" {
		return filepath.Join(repoRoot, "testdata", "regression", fmt.Sprintf("loop-%s.behavior-baseline.json", profileName))
	}
	if filepath.IsAbs(baselinePath) {
		return baselinePath
	}
	return filepath.Join(repoRoot, baselinePath)
}

func ResolveVerboseArtifacts(defaultValue, forceEnable, forceDisable bool) (bool, error) {
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

func ResolveRoleConfigPath(repoRoot, roleConfig string) string {
	if roleConfig == "" {
		return ""
	}
	if filepath.IsAbs(roleConfig) {
		return roleConfig
	}
	return filepath.Join(repoRoot, roleConfig)
}

func (h *LoopHandler) ExecuteLoopAction(action string, args []string, ctx harness.Context) (harness.CommandOutcome, error) {
	switch action {
	case "capabilities":
		return h.runLoopCapabilitiesCommand(args)
	case "doctor":
		return h.runLoopDoctorCommand(args)
	case "profiles":
		return h.runLoopProfilesCommand(args)
	case "quality":
		return h.runLoopProfileCommand("quality", args, ctx)
	case "regression":
		return h.runLoopRegressionCommand(args, ctx)
	case "profile":
		if len(args) == 0 {
			return harness.CommandOutcome{}, harness.NewFailure(
				harness.CodeUsage,
				"usage: agentcli loop profile <name> [--repo-root path] [--threshold score] [--max-iterations n] [--branch name] [--api url] [--role-config path] [--verbose-artifacts|--no-verbose-artifacts]",
				"",
				false,
			)
		}
		return h.runLoopProfileCommand(args[0], args[1:], ctx)
	case "lab":
		return h.runLoopLabCommand(args, ctx)
	case "run", "judge", "autofix":
		return h.runLoopClassicCommand(action, args, ctx)
	default:
		if out, err, ok := h.runLoopNamedProfileCommand(action, args, ctx); ok {
			return out, err
		}
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown loop action: %s", action),
			"use 'agentcli loop --format json capabilities' to discover commands",
			false,
		)
	}
}

func (h *LoopHandler) runLoopCapabilitiesCommand(args []string) (harness.CommandOutcome, error) {
	if len(args) > 0 {
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unexpected argument: %s", args[0]),
			"use global flags before action, for example: agentcli loop --format json capabilities",
			false,
		)
	}
	return harness.CommandOutcome{Data: harness.DefaultCapabilities()}, nil
}

func (h *LoopHandler) runLoopNamedProfileCommand(name string, args []string, ctx harness.Context) (harness.CommandOutcome, error, bool) {
	repoRoot, err := ParseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err), true
	}
	profiles, err := GetLoopProfiles(repoRoot)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "read loop profiles", "", false, err), true
	}
	if _, ok := profiles[name]; !ok {
		return harness.CommandOutcome{}, nil, false
	}
	out, runErr := h.runLoopProfileCommand(name, args, ctx)
	return out, runErr, true
}

func (h *LoopHandler) runLoopDoctorCommand(args []string) (harness.CommandOutcome, error) {
	opts, err := ParseLoopFlags(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	report := harnessloop.LoopDoctor(opts.RepoRoot)
	data := any(report)
	if opts.Markdown {
		data = map[string]any{"report": report, "markdown": harnessloop.RenderDoctorMarkdown(report)}
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

func (h *LoopHandler) runLoopProfilesCommand(args []string) (harness.CommandOutcome, error) {
	repoRoot, err := ParseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	profiles, err := GetLoopProfiles(repoRoot)
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
		lines = append(lines, FormatLoopProfile(name, profiles[name]))
	}

	return harness.CommandOutcome{
		Checks: []harness.CheckResult{{Name: "profiles_loaded", Status: harness.StatusOK, Details: fmt.Sprintf("%d profile(s)", len(names))}},
		Data: map[string]any{
			"repo_root": repoRoot,
			"profiles":  lines,
		},
	}, nil
}

func (h *LoopHandler) runLoopClassicCommand(action string, args []string, ctx harness.Context) (harness.CommandOutcome, error) {
	if action == "judge" {
		action = "run"
	}
	opts, err := ParseLoopFlags(args)
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
	if ctx.DryRun {
		return dryRunOutcome("loop "+action, map[string]any{"config": cfg}), nil
	}
	result, err := h.runLoop(opts.APIURL, action, cfg)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop run failed", "", false, err)
	}
	return OutcomeFromRunResult(result), nil
}

func (h *LoopHandler) runLoopProfileCommand(name string, args []string, ctx harness.Context) (harness.CommandOutcome, error) {
	repoRoot, err := ParseLoopProfilesRepoRoot(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	profiles, err := GetLoopProfiles(repoRoot)
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
	opts, err := ParseLoopQualityFlags(profile, args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	roleConfig := profile.RoleConfig
	if opts.RoleConfig != "" {
		roleConfig = opts.RoleConfig
	}
	roleConfigPath := roleConfig
	if opts.APIURL == "" {
		roleConfigPath = ResolveRoleConfigPath(opts.RepoRoot, roleConfig)
	}
	verboseArtifacts, err := ResolveVerboseArtifacts(profile.VerboseArtifacts, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	cfg := harnessloop.Config{
		RepoRoot:         opts.RepoRoot,
		Threshold:        opts.Threshold,
		MaxIterations:    opts.MaxIterations,
		Branch:           opts.Branch,
		Mode:             profile.Mode,
		RoleConfigPath:   roleConfigPath,
		Budget:           profile.Budget,
		VerboseArtifacts: verboseArtifacts,
	}
	if ctx.DryRun {
		return dryRunOutcome("loop profile "+name, map[string]any{"profile": name, "config": cfg}), nil
	}

	result, err := h.runLoop(opts.APIURL, "judge", cfg)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop profile run failed", "", false, err)
	}
	outcome := OutcomeFromRunResult(result)
	outcome.Checks = append([]harness.CheckResult{{Name: "profile", Status: harness.StatusOK, Details: name}}, outcome.Checks...)
	return outcome, nil
}

func (h *LoopHandler) runLoopRegressionCommand(args []string, ctx harness.Context) (harness.CommandOutcome, error) {
	regressionFlags, profileArgs, err := ParseLoopRegressionFlags(args)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	repoRoot, err := ParseLoopProfilesRepoRoot(profileArgs)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	profiles, err := GetLoopProfiles(repoRoot)
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

	opts, err := ParseLoopQualityFlags(profile, profileArgs)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	roleConfig := profile.RoleConfig
	if opts.RoleConfig != "" {
		roleConfig = opts.RoleConfig
	}
	roleConfigPath := roleConfig
	if opts.APIURL == "" {
		roleConfigPath = ResolveRoleConfigPath(opts.RepoRoot, roleConfig)
	}
	verboseArtifacts, err := ResolveVerboseArtifacts(profile.VerboseArtifacts, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}

	cfg := harnessloop.Config{
		RepoRoot:         opts.RepoRoot,
		Threshold:        opts.Threshold,
		MaxIterations:    opts.MaxIterations,
		Branch:           opts.Branch,
		Mode:             profile.Mode,
		RoleConfigPath:   roleConfigPath,
		Budget:           profile.Budget,
		VerboseArtifacts: verboseArtifacts,
	}
	if ctx.DryRun {
		return dryRunOutcome("loop regression", map[string]any{
			"profile":        regressionFlags.Profile,
			"baseline_path":  ResolveLoopRegressionBaselinePath(opts.RepoRoot, regressionFlags.Profile, regressionFlags.BaselinePath),
			"write_baseline": regressionFlags.WriteBaseline,
			"config":         cfg,
		}), nil
	}

	result, err := h.runLoop(opts.APIURL, "judge", cfg)
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop regression run failed", "", false, err)
	}

	snapshot := harnessloop.BuildBehaviorSnapshot(result)
	baselinePath := ResolveLoopRegressionBaselinePath(opts.RepoRoot, regressionFlags.Profile, regressionFlags.BaselinePath)
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
			Checks:    []harness.CheckResult{{Name: "baseline_written", Status: harness.StatusOK, Details: baselinePath}},
			Artifacts: []harness.Artifact{{Name: "behavior-baseline", Kind: "json", Path: baselinePath}},
			Data: LoopRegressionReport{
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
				Artifacts: []harness.Artifact{{Name: "behavior-baseline", Kind: "json", Path: baselinePath}},
				Data: LoopRegressionReport{
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
	report := LoopRegressionReport{
		SchemaVersion: "v1",
		Profile:       regressionFlags.Profile,
		BaselinePath:  baselinePath,
		RunID:         result.RunID,
		Pass:          len(drifts) == 0,
		DriftCount:    len(drifts),
		Drifts:        drifts,
	}
	outcome := harness.CommandOutcome{
		Checks: []harness.CheckResult{{
			Name:    "behavior_drift",
			Status:  boolStatus(len(drifts) == 0),
			Details: fmt.Sprintf("%d drift(s)", len(drifts)),
		}},
		Artifacts: []harness.Artifact{{Name: "behavior-baseline", Kind: "json", Path: baselinePath}},
		Data:      report,
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

func (h *LoopHandler) runLoopLabCommand(args []string, ctx harness.Context) (harness.CommandOutcome, error) {
	if len(args) == 0 {
		return harness.CommandOutcome{}, harness.NewFailure(harness.CodeUsage, "usage: agentcli loop lab [compare|replay|run|judge|autofix] ...", "", false)
	}
	action := args[0]
	opts, err := ParseLoopLabFlags(args[1:])
	if err != nil {
		return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeUsage, err.Error(), "", false, err)
	}
	if ctx.DryRun {
		return dryRunOutcome("loop lab "+action, map[string]any{"action": action, "flags": opts}), nil
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
		outcome := harness.CommandOutcome{Data: report}
		if path, err := harnessloop.WriteCompareOutput(opts.RepoRoot, report, opts.Format, opts.Out); err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeFileIO, "write compare report", "", false, err)
		} else if path != "" {
			outcome.Artifacts = append(outcome.Artifacts, harness.Artifact{Name: "compare-report", Kind: opts.Format, Path: path})
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
			Checks: []harness.CheckResult{{Name: "replay_pass", Status: boolStatus(report.ReplayJudge.Pass)}},
			Data:   report,
		}
		if !report.ReplayJudge.Pass {
			outcome.Failures = append(outcome.Failures, harness.Failure{Code: string(harness.CodeExecution), Message: "replay judge failed", Retryable: false})
		}
		return outcome, nil
	case "run", "judge", "autofix":
		roleConfigPath := opts.RoleConfig
		if opts.APIURL == "" {
			roleConfigPath = ResolveRoleConfigPath(opts.RepoRoot, opts.RoleConfig)
		}
		verboseArtifacts, err := ResolveVerboseArtifacts(false, opts.VerboseArtifacts, opts.NoVerboseArtifacts)
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
		result, err := h.runLoop(opts.APIURL, action, cfg)
		if err != nil {
			return harness.CommandOutcome{}, harness.WrapFailure(harness.CodeExecution, "loop lab run failed", "", false, err)
		}
		return OutcomeFromRunResult(result), nil
	default:
		return harness.CommandOutcome{}, harness.NewFailure(
			harness.CodeUsage,
			fmt.Sprintf("unknown lab action: %s", action),
			"use 'agentcli loop --format json capabilities' to discover lab actions",
			false,
		)
	}
}

func OutcomeFromRunResult(result harnessloop.RunResult) harness.CommandOutcome {
	outcome := harness.CommandOutcome{
		Checks: []harness.CheckResult{
			{Name: "scenario_ok", Status: boolStatus(result.Scenario.OK), Details: result.Scenario.Name},
			{Name: "judge_pass", Status: boolStatus(result.Judge.Pass), Details: fmt.Sprintf("score=%.2f threshold=%.2f", result.Judge.Score, result.Judge.Threshold)},
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

func dryRunOutcome(command string, details any) harness.CommandOutcome {
	return harness.CommandOutcome{
		Checks: []harness.CheckResult{{Name: "dry_run", Status: harness.StatusOK, Details: "execution skipped"}},
		Data: map[string]any{
			"dry_run": true,
			"command": command,
			"details": details,
		},
	}
}

func boolStatus(ok bool) harness.Status {
	if ok {
		return harness.StatusOK
	}
	return harness.StatusFail
}
