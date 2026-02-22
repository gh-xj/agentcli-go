package main

import (
	"fmt"
	"os"

	"github.com/gh-xj/gokit"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return gokit.ExitUsage
	}

	switch args[0] {
	case "new":
		return runNew(args[1:])
	case "add":
		return runAdd(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "-h", "--help", "help":
		printUsage()
		return gokit.ExitSuccess
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		printUsage()
		return gokit.ExitUsage
	}
}

func runNew(args []string) int {
	baseDir := "."
	module := ""
	name := ""
	runtime := "legacy"
	localGokitPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return gokit.ExitUsage
			}
			baseDir = args[i+1]
			i++
		case "--module":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--module requires a value")
				return gokit.ExitUsage
			}
			module = args[i+1]
			i++
		case "--runtime":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--runtime requires a value (legacy|cobrax)")
				return gokit.ExitUsage
			}
			runtime = args[i+1]
			i++
		case "--local-gokit":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--local-gokit requires a path")
				return gokit.ExitUsage
			}
			localGokitPath = args[i+1]
			i++
		default:
			if name == "" {
				name = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
				return gokit.ExitUsage
			}
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: gokit new [--dir path] [--module module/path] [--runtime legacy|cobrax] [--local-gokit path] <name>")
		return gokit.ExitUsage
	}

	root, err := gokit.ScaffoldNewWithOptions(baseDir, name, module, gokit.ScaffoldOptions{
		Runtime:        runtime,
		LocalGokitPath: localGokitPath,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return gokit.ExitFailure
	}
	fmt.Fprintf(os.Stdout, "created project: %s\n", root)
	return gokit.ExitSuccess
}

func runAdd(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gokit add command [--dir path] <name>")
		return gokit.ExitUsage
	}
	if args[0] != "command" {
		fmt.Fprintf(os.Stderr, "unknown add target: %s\n", args[0])
		return gokit.ExitUsage
	}

	rootDir := "."
	name := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return gokit.ExitUsage
			}
			rootDir = args[i+1]
			i++
		default:
			if name == "" {
				name = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
				return gokit.ExitUsage
			}
		}
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "usage: gokit add command [--dir path] <name>")
		return gokit.ExitUsage
	}
	if err := gokit.ScaffoldAddCommand(rootDir, name); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return gokit.ExitFailure
	}
	fmt.Fprintf(os.Stdout, "added command: %s\n", name)
	return gokit.ExitSuccess
}

func runDoctor(args []string) int {
	rootDir := "."
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--dir requires a value")
				return gokit.ExitUsage
			}
			rootDir = args[i+1]
			i++
		case "--json":
			jsonOutput = true
		default:
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", args[i])
			return gokit.ExitUsage
		}
	}

	report := gokit.Doctor(rootDir)
	if jsonOutput {
		out, err := report.JSON()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return gokit.ExitFailure
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
		return gokit.ExitSuccess
	}
	return gokit.ExitFailure
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "gokit scaffold CLI")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  gokit new [--dir path] [--module module/path] [--runtime legacy|cobrax] [--local-gokit path] <name>")
	fmt.Fprintln(os.Stderr, "  gokit add command [--dir path] <name>")
	fmt.Fprintln(os.Stderr, "  gokit doctor [--dir path] [--json]")
}
