package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/internal/migrate"
)

func runMigrate(args []string) int {
	source := "scripts"
	mode := migrate.ModeSafe
	dryRun := false
	apply := false
	outputRoot := ""

	if len(args) == 1 {
		switch args[0] {
		case "-h", "--help", "help":
			printMigrateUsage()
			return agentcli.ExitSuccess
		}
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--source requires a value")
				return agentcli.ExitUsage
			}
			source = args[i+1]
			i++
		case "--mode":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--mode requires a value")
				return agentcli.ExitUsage
			}
			mode = migrate.MigrationMode(args[i+1])
			i++
		case "--dry-run":
			dryRun = true
		case "--apply":
			apply = true
		case "--out":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--out requires a value")
				return agentcli.ExitUsage
			}
			outputRoot = args[i+1]
			i++
		default:
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
			return agentcli.ExitUsage
		}
	}

	if apply && dryRun {
		fmt.Fprintln(os.Stderr, "cannot use --dry-run and --apply together")
		return agentcli.ExitUsage
	}
	if !apply && !dryRun {
		dryRun = true
	}
	if mode != migrate.ModeSafe && mode != migrate.ModeInPlace {
		fmt.Fprintln(os.Stderr, "--mode must be one of: safe, in-place")
		return agentcli.ExitUsage
	}
	if strings.TrimSpace(source) == "" {
		fmt.Fprintln(os.Stderr, "--source requires a non-empty value")
		return agentcli.ExitUsage
	}

	scanResult, err := migrate.ScanScripts(".", source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan scripts: %v\n", err)
		return agentcli.ExitFailure
	}
	plan := migrate.BuildPlan(scanResult)

	if dryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		fmt.Fprintln(os.Stdout, "migration plan (dry-run)")
		if err := enc.Encode(plan.Summary); err != nil {
			fmt.Fprintf(os.Stderr, "encode migration summary: %v\n", err)
			return agentcli.ExitFailure
		}
		return agentcli.ExitSuccess
	}

	result, err := migrate.Generate(plan, migrate.GenerateOptions{
		Mode:       mode,
		OutputRoot: outputRoot,
		Apply:      true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate migration output: %v\n", err)
		return agentcli.ExitFailure
	}
	fmt.Fprintf(os.Stdout, "migration generated at: %s (commands=%d)\n", result.OutputRoot, result.GeneratedCommands)
	return agentcli.ExitSuccess
}

func printMigrateUsage() {
	fmt.Fprintln(os.Stderr, "usage: agentcli migrate --source path [--mode safe|in-place] [--dry-run|--apply] [--out path]")
	fmt.Fprintln(os.Stderr, "  default mode: safe")
	fmt.Fprintln(os.Stderr, "  source defaults to: scripts")
	fmt.Fprintln(os.Stderr, "  examples:")
	fmt.Fprintln(os.Stderr, "    agentcli migrate --source ./scripts --mode safe --dry-run")
	fmt.Fprintln(os.Stderr, "    agentcli migrate --source ./scripts --mode safe --apply")
}
