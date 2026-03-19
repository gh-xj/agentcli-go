package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/service"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return agentcli.ExitUsage
	}

	switch args[0] {
	case "--version", "-v":
		printVersion()
		return agentcli.ExitSuccess
	case "new":
		return runNew(args[1:])
	case "add":
		return runAdd(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "migrate":
		return runMigrate(args[1:])
	case "loop":
		return runLoop(args[1:])
	case "loop-server":
		return runLoopServer(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return agentcli.ExitSuccess
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		printUsage()
		return agentcli.ExitUsage
	}
}

func printVersion() {
	v, c, d := versionInfo()
	fmt.Fprintf(os.Stdout, "agentcli %s (%s %s)\n", v, c, d)
}

func versionInfo() (string, string, string) {
	outVersion := version
	outCommit := commit
	outDate := date

	if info, ok := debug.ReadBuildInfo(); ok {
		if outVersion == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			outVersion = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if outCommit == "none" && setting.Value != "" {
					outCommit = setting.Value
				}
			case "vcs.time":
				if outDate == "unknown" && setting.Value != "" {
					outDate = strings.TrimSpace(strings.SplitN(setting.Value, " ", 2)[0])
				}
			}
		}
	}
	if outVersion == "" {
		outVersion = "dev"
	}
	return outVersion, outCommit, outDate
}

func runNew(args []string) int {
	baseDir := "."
	module := ""
	name := ""
	inExistingModule := false
	minimal := false
	full := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return agentcli.ExitUsage
			}
			baseDir = args[i+1]
			i++
		case "--module":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--module requires a value")
				return agentcli.ExitUsage
			}
			module = args[i+1]
			i++
		case "--in-existing-module":
			inExistingModule = true
		case "--minimal":
			minimal = true
		case "--full":
			full = true
		default:
			if name == "" {
				name = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
				return agentcli.ExitUsage
			}
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: agentcli new [--dir path] [--in-existing-module] [--minimal] [--full] [--module module/path] <name>")
		return agentcli.ExitUsage
	}
	if inExistingModule && module != "" {
		fmt.Fprintln(os.Stderr, "--module cannot be used with --in-existing-module")
		return agentcli.ExitUsage
	}
	if minimal && full {
		fmt.Fprintln(os.Stderr, "--minimal and --full cannot be used together")
		return agentcli.ExitUsage
	}

	root, err := service.Get().ScaffoldSvc.New(baseDir, name, module, service.ScaffoldNewOptions{
		InExistingModule: inExistingModule,
		Minimal:          minimal,
		Full:             full,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitFailure
	}
	fmt.Fprintf(os.Stdout, "created project: %s\n", root)
	return agentcli.ExitSuccess
}

func runAdd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: agentcli add command [--dir path] [--description text] [--preset name] [--list-presets] <name>")
		return agentcli.ExitUsage
	}
	if args[0] != "command" {
		fmt.Fprintf(os.Stderr, "unknown add target: %s\n", args[0])
		return agentcli.ExitUsage
	}

	rootDir := "."
	name := ""
	description := ""
	preset := ""
	listPresets := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return agentcli.ExitUsage
			}
			rootDir = args[i+1]
			i++
		case "--description":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--description requires a value")
				return agentcli.ExitUsage
			}
			description = args[i+1]
			i++
		case "--preset":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--preset requires a value")
				return agentcli.ExitUsage
			}
			preset = args[i+1]
			i++
		case "--list-presets":
			listPresets = true
		default:
			if name == "" {
				name = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
				return agentcli.ExitUsage
			}
		}
	}

	if listPresets {
		for _, presetName := range service.CommandPresetNames() {
			desc, _ := service.CommandPresetDescription(presetName)
			fmt.Fprintf(os.Stdout, "%s: %s\n", presetName, desc)
		}
		return agentcli.ExitSuccess
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: agentcli add command [--dir path] [--description text] [--preset name] [--list-presets] <name>")
		return agentcli.ExitUsage
	}
	if err := service.Get().ScaffoldSvc.AddCommand(rootDir, name, description, preset); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return agentcli.ExitFailure
	}
	fmt.Fprintf(os.Stdout, "added command: %s\n", name)
	return agentcli.ExitSuccess
}

func runDoctor(args []string) int {
	rootDir := "."
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return agentcli.ExitUsage
			}
			rootDir = args[i+1]
			i++
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
			return agentcli.ExitUsage
		}
	}

	report := service.Get().DoctorSvc.Run(rootDir)
	if jsonOutput {
		out, err := report.JSON()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return agentcli.ExitFailure
		}
		fmt.Fprintln(os.Stdout, out)
	} else {
		if report.OK {
			fmt.Fprintln(os.Stdout, "doctor: ok")
		} else {
			fmt.Fprintln(os.Stdout, "doctor: failed")
			for _, f := range report.Findings {
				fmt.Fprintf(os.Stdout, "- [%s] %s: %s\n", f.Code, f.Path, f.Message)
			}
		}
	}

	if report.OK {
		return agentcli.ExitSuccess
	}
	return agentcli.ExitFailure
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "agentcli scaffold CLI (from agentcli-go)")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  agentcli new [--dir path] [--in-existing-module] [--minimal] [--full] [--module module/path] <name>")
	fmt.Fprintln(os.Stderr, "    monorepo default recommendation: use --in-existing-module")
	fmt.Fprintln(os.Stderr, "  agentcli add command [--dir path] [--description text] [--preset name] [--list-presets] <name>")
	fmt.Fprintln(os.Stderr, "  agentcli doctor [--dir path] [--json]")
	fmt.Fprintln(os.Stderr, "  agentcli --version")
	fmt.Fprintln(os.Stderr, "  agentcli migrate --source path [--mode safe|in-place] [--dry-run|--apply] [--out path]")
	fmt.Fprintln(os.Stderr, "    agent prompt: run 'agentcli migrate --source ./scripts --mode safe --dry-run' first")
	fmt.Fprintln(os.Stderr, "  agentcli loop [global flags] [run|judge|autofix|doctor|quality|profiles|profile|<profile>|regression|capabilities|lab] [command flags]")
	fmt.Fprintln(os.Stderr, "    global flags: [--format text|json|ndjson] [--summary path] [--no-color] [--dry-run] [--explain]")
	fmt.Fprintln(os.Stderr, "  agentcli loop lab [compare|replay|run|judge|autofix] [advanced flags]")
	fmt.Fprintln(os.Stderr, "  agentcli loop-server [--addr host:port] [--repo-root path]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Presets for add command:")
	for _, presetName := range service.CommandPresetNames() {
		desc, _ := service.CommandPresetDescription(presetName)
		fmt.Fprintf(os.Stderr, "  %s: %s\n", presetName, desc)
	}
}
