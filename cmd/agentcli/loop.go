//go:build !agentcli_core
// +build !agentcli_core

package main

import (
	"fmt"
	"os"
	"time"

	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
	"github.com/gh-xj/agentcli-go/internal/loopapi"
	harness "github.com/gh-xj/agentcli-go/tools/harness"
	loopcommands "github.com/gh-xj/agentcli-go/tools/harness/commands"
)

func runLoop(args []string) int {
	runtime := loopcommands.LoopRuntimeFlags{Format: "text"}
	if len(args) == 0 {
		return emitLoopFailureSummary(
			"loop",
			runtime,
			harness.NewFailure(
				harness.CodeUsage,
				"usage: agentcli loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]",
				"",
				false,
			),
		)
	}

	parsedRuntime, remaining, err := loopcommands.ParseLoopRuntimeFlags(args)
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
				"usage: agentcli loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]",
				"",
				false,
			),
		)
	}

	handler := loopcommands.NewLoopHandler(runLoopWithOptionalAPI)
	action := remaining[0]
	actionArgs := remaining[1:]
	summary, execErr := harness.Run(harness.CommandInput{
		Name:        "loop " + action,
		SummaryPath: runtime.SummaryPath,
		DryRun:      runtime.DryRun,
		Explain:     runtime.Explain,
		Execute: func(ctx harness.Context) (harness.CommandOutcome, error) {
			return handler.ExecuteLoopAction(action, actionArgs, ctx)
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

func emitLoopFailureSummary(command string, runtime loopcommands.LoopRuntimeFlags, err error) int {
	now := time.Now().UTC()
	summary := harness.CommandSummary{
		SchemaVersion: harness.SummarySchemaVersion,
		Command:       command,
		Status:        harness.StatusFail,
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
