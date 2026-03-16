package service

// commandPresets maps preset names to default descriptions.
var commandPresets = map[string]string{
	"file-sync":                "sync files between source and destination",
	"http-client":              "send HTTP requests to a target endpoint",
	"deploy-helper":            "run deterministic deploy workflow checks",
	"task-replay-orchestrator": "orchestrate external repo task runs with env injection and timeout hooks",
}

// ──────────────────────────────────────────────────────────────────────
// Existing scaffold templates (moved from root scaffold.go)
// ──────────────────────────────────────────────────────────────────────

const goModTpl = `module {{.Module}}

go 1.25.5

require github.com/gh-xj/agentcli-go v0.3.0

{{.GokitReplaceLine}}
`

const mainTpl = `package main

import (
	"os"

	"{{.Module}}/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
`

const rootCmdTpl = `package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/cobrax"
)

type command struct {
	Description string
	Run         func(*agentcli.AppContext, []string) error
}

var commandRegistry = map[string]command{}

func init() {
	registerBuiltins()
	// agentcli:add-command
}

func registerCommand(name string, cmd command) {
	commandRegistry[name] = cmd
}

func registerBuiltins() {
	registerCommand("version", command{
		Description: "print build metadata",
		Run: func(app *agentcli.AppContext, _ []string) error {
			data := map[string]string{
				"schema_version": "v1",
				"name":           "{{.Name}}",
				"version":        "dev",
				"commit":         "none",
				"date":           "unknown",
			}
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(data)
			}
			_, err := fmt.Fprintf(os.Stdout, "%s %s (%s %s)\n", data["name"], data["version"], data["commit"], data["date"])
			return err
		},
	})
}

func Execute(args []string) int {
	commands := make([]cobrax.CommandSpec, 0, len(commandRegistry))
	names := make([]string, 0, len(commandRegistry))
	for name := range commandRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := commandRegistry[name]
		commands = append(commands, cobrax.CommandSpec{
			Use:   name,
			Short: cmd.Description,
			Run:   cmd.Run,
		})
	}

	return cobrax.Execute(cobrax.RootSpec{
		Use:   "{{.Name}}",
		Short: "{{.Name}} CLI",
		Meta: agentcli.AppMeta{
			Name:    "{{.Name}}",
			Version: "dev",
			Commit:  "none",
			Date:    "unknown",
		},
		Commands: commands,
	}, args)
}
`

const addCommandTpl = `package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	{{- if eq .Preset "task-replay-orchestrator" }}
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
	{{- end }}

	"github.com/gh-xj/agentcli-go"
)

func init() {
}

func {{.Module}}Command() command {
	return command{
		Description: {{printf "%q" .Description}},
		Run: func(app *agentcli.AppContext, args []string) error {
			preset := {{printf "%q" .Preset}}
			if preset == "" {
				preset = "custom"
			}
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"command": "{{.Name}}",
					"preset":  preset,
					"ok":      true,
					"args":    len(args),
				})
			}
			{{- if eq .Preset "file-sync" }}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} preset=file-sync: synced %d items\n", len(args))
			{{- else if eq .Preset "http-client" }}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} preset=http-client: request plan ready with %d args\n", len(args))
			{{- else if eq .Preset "deploy-helper" }}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} preset=deploy-helper: deploy checks passed for %d args\n", len(args))
			{{- else if eq .Preset "task-replay-orchestrator" }}
			repoRoot := ""
			taskName := "replay:emit"
			timeout := 10 * time.Minute
			timeoutHook := ""
			envPairs := make(map[string]string)
			for i := 0; i < len(args); i++ {
				switch args[i] {
				case "--repo":
					if i+1 >= len(args) {
						return fmt.Errorf("--repo requires a value")
					}
					repoRoot = args[i+1]
					i++
				case "--task":
					if i+1 >= len(args) {
						return fmt.Errorf("--task requires a value")
					}
					taskName = args[i+1]
					i++
				case "--env":
					if i+1 >= len(args) {
						return fmt.Errorf("--env requires KEY=VALUE")
					}
					parts := strings.SplitN(args[i+1], "=", 2)
					if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
						return fmt.Errorf("--env requires KEY=VALUE")
					}
					envPairs[parts[0]] = parts[1]
					i++
				case "--timeout":
					if i+1 >= len(args) {
						return fmt.Errorf("--timeout requires a value")
					}
					d, parseErr := time.ParseDuration(args[i+1])
					if parseErr != nil || d <= 0 {
						return fmt.Errorf("invalid --timeout value: %s", args[i+1])
					}
					timeout = d
					i++
				case "--timeout-hook":
					if i+1 >= len(args) {
						return fmt.Errorf("--timeout-hook requires a value")
					}
					timeoutHook = args[i+1]
					i++
				default:
					return fmt.Errorf("unexpected argument: %s (usage: --repo <path> [--task <name>] [--env KEY=VALUE]... [--timeout 30s] [--timeout-hook '<cmd>'])", args[i])
				}
			}
			if strings.TrimSpace(repoRoot) == "" {
				return fmt.Errorf("--repo is required")
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			cmd := exec.CommandContext(ctx, "task", "-d", repoRoot, taskName)
			cmd.Env = os.Environ()
			for k, v := range envPairs {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
			out, runErr := cmd.CombinedOutput()
			timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
			hookOutput := ""
			if timedOut && strings.TrimSpace(timeoutHook) != "" {
				hook := exec.Command("sh", "-c", timeoutHook)
				hook.Dir = repoRoot
				hook.Env = append(os.Environ(),
					"AGENTCLI_TIMEOUT_REPO="+repoRoot,
					"AGENTCLI_TIMEOUT_TASK="+taskName,
					"AGENTCLI_TIMEOUT="+timeout.String(),
				)
				hookOut, hookErr := hook.CombinedOutput()
				hookOutput = strings.TrimSpace(string(hookOut))
				if hookErr != nil {
					return fmt.Errorf("task timed out after %s; timeout hook failed: %w\ntask output:\n%s\nhook output:\n%s", timeout, hookErr, strings.TrimSpace(string(out)), hookOutput)
				}
			}
			if runErr != nil {
				if timedOut {
					if strings.TrimSpace(timeoutHook) != "" {
						return fmt.Errorf("task timed out after %s (timeout hook executed)\n%s", timeout, strings.TrimSpace(string(out)))
					}
					return fmt.Errorf("task timed out after %s\n%s", timeout, strings.TrimSpace(string(out)))
				}
				return fmt.Errorf("task wrapper failed: %w\n%s", runErr, strings.TrimSpace(string(out)))
			}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} preset={{.Preset}}: repo=%s task=%s env=%d timeout=%s\n", repoRoot, taskName, len(envPairs), timeout)
			if err == nil && len(out) > 0 {
				_, err = fmt.Fprint(os.Stdout, string(out))
			}
			if err == nil && hookOutput != "" {
				_, err = fmt.Fprintf(os.Stdout, "\n# timeout-hook\n%s\n", hookOutput)
			}
			{{- else }}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} executed with %d args\n", len(args))
			{{- end }}
			return err
		},
	}
}
`

const appBootstrapTpl = `package app

import (
	"os"
	"slices"

	"github.com/gh-xj/agentcli-go/dal"
)

func Bootstrap() {
	verbose := slices.Contains(os.Args, "-v") || slices.Contains(os.Args, "--verbose")
	dal.NewLogger().Init(verbose, os.Stderr)
}
`

const appLifecycleTpl = `package app

type Hooks struct{}

func (Hooks) Preflight() error {
	return nil
}

func (Hooks) Postflight() error {
	return nil
}
`

const appErrorsTpl = `package app

import "fmt"

func UsageError(message string) error {
	return fmt.Errorf("usage: %s", message)
}
`

const configSchemaTpl = `package config

type Config struct {
	Environment string ` + "`json:\"environment\"`" + `
}
`

const configLoadTpl = `package config

func Load() Config {
	return Config{Environment: "default"}
}
`

const outputTpl = `package appio

import (
	"encoding/json"
	"io"
)

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
`

const versionTpl = `package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)
`

const e2eTestTpl = `package e2e

import "testing"

func TestPlaceholder(t *testing.T) {
	t.Skip("add end-to-end command tests")
}
`

const smokeSchemaTpl = `{
  "schema_version": "v1",
  "required_keys": [
    "schema_version",
    "name",
    "version",
    "commit",
    "date"
  ]
}
`

const smokeCheckTpl = `package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

type schema struct {
	SchemaVersion string   ` + "`json:\"schema_version\"`" + `
	RequiredKeys  []string ` + "`json:\"required_keys\"`" + `
}

func main() {
	schemaPath := flag.String("schema", "", "path to schema file")
	inputPath := flag.String("input", "", "path to smoke output json")
	flag.Parse()

	if *schemaPath == "" || *inputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: smokecheck --schema <schema.json> --input <output.json>")
		os.Exit(2)
	}
	if err := run(*schemaPath, *inputPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("smoke schema check passed")
}

func run(schemaPath, inputPath string) error {
	s, err := readSchema(schemaPath)
	if err != nil {
		return err
	}
	payload, err := readPayload(inputPath)
	if err != nil {
		return err
	}
	for _, key := range s.RequiredKeys {
		if _, ok := payload[key]; !ok {
			return fmt.Errorf("missing required key: %s", key)
		}
	}
	if got, _ := payload["schema_version"].(string); got != s.SchemaVersion {
		return fmt.Errorf("schema_version mismatch: got %q want %q", got, s.SchemaVersion)
	}
	return nil
}

func readSchema(path string) (schema, error) {
	var out schema
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	if out.SchemaVersion == "" {
		return out, errors.New("schema_version is required in schema")
	}
	if len(out.RequiredKeys) == 0 {
		return out, errors.New("required_keys must not be empty")
	}
	return out, nil
}

func readPayload(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
`

const taskfileTpl = `version: "3"

tasks:
  deps:
    desc: Sync module dependencies
    cmds:
      - go mod tidy

  fmt:
    desc: Format Go files
    cmds:
      - gofmt -w $(find . -name '*.go' -type f)

  fmt:check:
    desc: Fail when gofmt would change files
    cmds:
      - test -z "$(gofmt -l $(find . -name '*.go' -type f))"

  lint:
    desc: Run static checks
    deps: [deps, fmt:check]
    cmds:
      - go vet ./...

  test:
    desc: Run tests
    deps: [deps, fmt:check]
    cmds:
      - go test ./...

  build:
    desc: Build binary
    deps: [deps, fmt:check]
    cmds:
      - mkdir -p bin
      - go build -o bin/{{.Name}} .

  smoke:
    desc: Deterministic smoke checks
    deps: [build]
    cmds:
      - mkdir -p test/smoke
      - rm -f test/smoke/version.output.json
      - ./bin/{{.Name}} --json version > test/smoke/version.output.json
      - go run ./internal/tools/smokecheck --schema test/smoke/version.schema.json --input test/smoke/version.output.json

  ci:
    desc: Canonical CI contract command
    deps: [lint, test, build, smoke]
    cmds:
      - echo "ci checks passed"

  verify:
    desc: Local aggregate verification
    deps: [ci]
    cmds:
      - echo "verify checks passed"
`

const readmeTpl = `# {{.Name}}

Generated by agentcli scaffold.

## Commands

- ` + "`version`" + `: print build metadata

## Verification

- ` + "`task ci`" + `: canonical CI command
- ` + "`task verify`" + `: local aggregate verification
`

const minimalReadmeTpl = `# {{.Name}}

Generated by agentcli scaffold (` + "`--minimal`" + ` mode).

This mode intentionally emits only a tiny runnable surface:

- ` + "`main.go`" + `
- ` + "`cmd/root.go`" + `
- ` + "`go.mod`" + ` / ` + "`go.sum`" + ` (when not using ` + "`--in-existing-module`" + `)
`

// ──────────────────────────────────────────────────────────────────────
// New DAG scaffold templates
// ──────────────────────────────────────────────────────────────────────

const scaffoldContainerTpl = `package service

type Container struct {
	// Add your services here
}

func NewContainer() *Container {
	return &Container{}
}

var globalContainer *Container

func Get() *Container {
	if globalContainer == nil {
		globalContainer = InitializeContainer()
	}
	return globalContainer
}

func Reset() {
	globalContainer = nil
}
`

const scaffoldWireTpl = `//go:build wireinject

package service

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewContainer,
)

func InitializeContainer() *Container {
	wire.Build(ProviderSet)
	return nil
}
`

const scaffoldDALInterfacesTpl = `package dal

// Add your data access interfaces here.
// Example:
// type FileSystem interface {
//     ReadFile(path string) ([]byte, error)
// }
`

const scaffoldDALFilesystemTpl = `package dal

// Add your data access implementations here.
`

const scaffoldOperatorInterfacesTpl = `package operator

// Add your business logic interfaces here.
// Example:
// type MyOperator interface {
//     Process(ctx context.Context, input string) (string, error)
// }
`

const scaffoldOperatorExampleTpl = `package operator

// TODO: Implement your operators here.
// Example:
// type MyOperatorImpl struct {
//     fs dal.FileSystem
// }
`
