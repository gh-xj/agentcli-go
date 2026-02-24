package agentcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
)

const rootCommandMarker = "// agentcli:add-command"

var validCommandName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var commandPresets = map[string]string{
	"file-sync":                "sync files between source and destination",
	"http-client":              "send HTTP requests to a target endpoint",
	"deploy-helper":            "run deterministic deploy workflow checks",
	"task-replay-emit-wrapper": "run task replay emit wrapper in external repo with env injection",
}

type templateData struct {
	Module           string
	Name             string
	Description      string
	Preset           string
	GokitReplaceLine string
}

// DoctorFinding describes a single compliance issue in a scaffolded project.
type DoctorFinding struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// DoctorReport summarizes scaffold compliance checks.
type DoctorReport struct {
	SchemaVersion string          `json:"schema_version"`
	OK            bool            `json:"ok"`
	Findings      []DoctorFinding `json:"findings"`
}

type ScaffoldNewOptions struct {
	InExistingModule bool
}

func (r DoctorReport) JSON() (string, error) {
	out, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ScaffoldNew creates a new CLI project using the golden agentcli layout.
// This is now cobrax-only by design.
func ScaffoldNew(baseDir, name, module string) (string, error) {
	return ScaffoldNewWithOptions(baseDir, name, module, ScaffoldNewOptions{})
}

// ScaffoldNewWithOptions creates a new CLI project using the golden agentcli
// layout with optional generation modes.
func ScaffoldNewWithOptions(baseDir, name, module string, opts ScaffoldNewOptions) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("project name is required")
	}
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	module = strings.TrimSpace(module)
	writeGoMod := true

	root := filepath.Join(baseDir, name)
	if err := ensureEmptyDir(root); err != nil {
		return "", err
	}

	if opts.InExistingModule {
		if module != "" {
			return "", errors.New("--module cannot be used with --in-existing-module")
		}
		modulePath, moduleRoot, err := resolveParentModule(root)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(moduleRoot, root)
		if err != nil {
			return "", fmt.Errorf("resolve path under parent module: %w", err)
		}
		module = modulePath
		if rel != "." {
			module = module + "/" + filepath.ToSlash(rel)
		}
		writeGoMod = false
	} else if module == "" {
		module = name
	}

	files := map[string]string{
		"main.go":                           mainTpl,
		"cmd/root.go":                       rootCmdTpl,
		"internal/app/bootstrap.go":         appBootstrapTpl,
		"internal/app/lifecycle.go":         appLifecycleTpl,
		"internal/app/errors.go":            appErrorsTpl,
		"internal/config/schema.go":         configSchemaTpl,
		"internal/config/load.go":           configLoadTpl,
		"internal/io/output.go":             outputTpl,
		"internal/tools/smokecheck/main.go": smokeCheckTpl,
		"pkg/version/version.go":            versionTpl,
		"test/e2e/cli_test.go":              e2eTestTpl,
		"test/smoke/version.schema.json":    smokeSchemaTpl,
		"Taskfile.yml":                      taskfileTpl,
		"README.md":                         readmeTpl,
	}
	if writeGoMod {
		files["go.mod"] = goModTpl
	}
	cliName := filepath.Base(strings.TrimSpace(name))
	if cliName == "" || cliName == "." || cliName == string(filepath.Separator) {
		return "", errors.New("project name is required")
	}

	d := templateData{
		Module:           module,
		Name:             cliName,
		GokitReplaceLine: detectLocalGokitReplaceLine(),
	}
	for path, body := range files {
		if err := writeTemplate(filepath.Join(root, path), body, d); err != nil {
			return "", err
		}
	}
	if writeGoMod {
		if err := runGoModTidy(root); err != nil {
			return "", err
		}
	}
	return root, nil
}

// ScaffoldAddCommand creates a command file and wires it into cmd/root.go.
func ScaffoldAddCommand(rootDir, commandName, description, preset string) error {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	if !validCommandName.MatchString(commandName) {
		return fmt.Errorf("invalid command name %q: use kebab-case [a-z0-9-]", commandName)
	}
	description = strings.TrimSpace(description)
	preset = strings.TrimSpace(preset)
	if preset != "" {
		presetDescription, ok := commandPresets[preset]
		if !ok {
			return fmt.Errorf("invalid preset %q: valid presets are %s", preset, strings.Join(sortedPresetNames(), ", "))
		}
		if description == "" {
			description = presetDescription
		}
	}
	if description == "" {
		description = fmt.Sprintf("describe %s", commandName)
	}

	cmdFile := filepath.Join(rootDir, "cmd", commandName+".go")
	if FileExists(cmdFile) {
		return fmt.Errorf("command file already exists: %s", cmdFile)
	}
	funcName := kebabToCamel(commandName)
	if err := writeTemplate(cmdFile, addCommandTpl, templateData{
		Name:        commandName,
		Module:      funcName,
		Description: description,
		Preset:      preset,
	}); err != nil {
		return err
	}

	rootFile := filepath.Join(rootDir, "cmd", "root.go")
	content, err := os.ReadFile(rootFile)
	if err != nil {
		return err
	}
	registerLine := fmt.Sprintf("registerCommand(%q, %sCommand())", commandName, funcName)
	text := string(content)
	if strings.Contains(text, registerLine) {
		return nil
	}
	idx := strings.Index(text, rootCommandMarker)
	if idx < 0 {
		return fmt.Errorf("marker %q not found in %s", rootCommandMarker, rootFile)
	}
	updated := text[:idx] + registerLine + "\n\t" + text[idx:]
	return os.WriteFile(rootFile, []byte(updated), 0644)
}

// CommandPresetNames returns the supported preset names in sorted order.
func CommandPresetNames() []string {
	return sortedPresetNames()
}

// CommandPresetDescription returns the description for a preset.
func CommandPresetDescription(name string) (string, bool) {
	description, ok := commandPresets[name]
	return description, ok
}

func sortedPresetNames() []string {
	names := make([]string, 0, len(commandPresets))
	for name := range commandPresets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// Doctor checks whether a project follows the golden scaffold contract.
func Doctor(rootDir string) DoctorReport {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}

	required := []string{
		"main.go",
		"cmd/root.go",
		"internal/app/bootstrap.go",
		"internal/app/lifecycle.go",
		"internal/app/errors.go",
		"internal/config/schema.go",
		"internal/config/load.go",
		"internal/io/output.go",
		"internal/tools/smokecheck/main.go",
		"pkg/version/version.go",
		"test/e2e/cli_test.go",
		"test/smoke/version.schema.json",
		"Taskfile.yml",
	}

	report := DoctorReport{SchemaVersion: "v1", OK: true, Findings: make([]DoctorFinding, 0)}

	if FileExists(filepath.Join(rootDir, "go.mod")) {
		required = append(required, "go.mod")
	} else if _, _, err := resolveParentModule(rootDir); err != nil {
		report.OK = false
		report.Findings = append(report.Findings, DoctorFinding{
			Code:    "missing_file",
			Path:    "go.mod",
			Message: "required file is missing and no parent module was found",
		})
	}

	for _, p := range required {
		abs := filepath.Join(rootDir, p)
		if !FileExists(abs) {
			report.OK = false
			report.Findings = append(report.Findings, DoctorFinding{Code: "missing_file", Path: p, Message: "required file is missing"})
		}
	}

	checkContains := func(relPath, code, want, msg string) {
		content, err := os.ReadFile(filepath.Join(rootDir, relPath))
		if err != nil {
			return
		}
		if !strings.Contains(string(content), want) {
			report.OK = false
			report.Findings = append(report.Findings, DoctorFinding{Code: code, Path: relPath, Message: msg})
		}
	}

	checkContains("cmd/root.go", "missing_contract", `"github.com/gh-xj/agentcli-go/cobrax"`, "cobrax runtime import missing")
	checkContains("cmd/root.go", "missing_contract", rootCommandMarker, "missing scaffold command marker")
	checkContains("Taskfile.yml", "missing_contract", "ci:", "canonical CI task missing")
	checkContains("Taskfile.yml", "missing_contract", "verify:", "local verification task missing")
	checkContains("Taskfile.yml", "missing_contract", "test/smoke/version.output.json", "smoke artifact output path missing")
	checkContains("Taskfile.yml", "missing_contract", "internal/tools/smokecheck", "smoke schema validation command missing")
	checkContains("internal/app/lifecycle.go", "missing_contract", "Preflight", "lifecycle preflight hook missing")
	checkContains("internal/app/lifecycle.go", "missing_contract", "Postflight", "lifecycle postflight hook missing")
	checkContains("test/smoke/version.schema.json", "missing_contract", "\"schema_version\": \"v1\"", "smoke schema version missing")

	slices.SortFunc(report.Findings, func(a, b DoctorFinding) int {
		if c := strings.Compare(a.Path, b.Path); c != 0 {
			return c
		}
		return strings.Compare(a.Code, b.Code)
	})
	report.OK = len(report.Findings) == 0
	return report
}

func ensureEmptyDir(root string) error {
	if FileExists(root) {
		entries, err := os.ReadDir(root)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("target directory is not empty: %s", root)
		}
		return nil
	}
	return os.MkdirAll(root, 0755)
}

func runGoModTidy(root string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run go mod tidy: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func resolveParentModule(targetRoot string) (modulePath, moduleRoot string, err error) {
	dir := filepath.Clean(targetRoot)
	for {
		modFile := filepath.Join(dir, "go.mod")
		raw, readErr := os.ReadFile(modFile)
		if readErr == nil {
			module := parseModulePath(string(raw))
			if module == "" {
				return "", "", fmt.Errorf("invalid go.mod in parent module root: %s", modFile)
			}
			return module, dir, nil
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return "", "", fmt.Errorf("read parent go.mod: %w", readErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", fmt.Errorf("parent go.mod not found for --in-existing-module target: %s", targetRoot)
}

func parseModulePath(goMod string) string {
	lines := strings.Split(goMod, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "module ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
		value = strings.Trim(value, `"`)
		return value
	}
	return ""
}

func writeTemplate(path, body string, data templateData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tpl, err := template.New(filepath.Base(path)).Parse(body)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

func kebabToCamel(in string) string {
	parts := strings.Split(in, "-")
	for i := range parts {
		if len(parts[i]) == 0 {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func detectLocalGokitReplaceLine() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 16; i++ {
		modFile := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modFile)
		if err == nil && strings.Contains(string(data), "module github.com/gh-xj/agentcli-go") {
			return fmt.Sprintf("replace github.com/gh-xj/agentcli-go => %s", dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

const goModTpl = `module {{.Module}}

go 1.25.5

require github.com/gh-xj/agentcli-go v0.2.2

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
	{{- if eq .Preset "task-replay-emit-wrapper" }}
	"os/exec"
	"strings"
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
			{{- else if eq .Preset "task-replay-emit-wrapper" }}
			repoRoot := ""
			taskName := "replay:emit"
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
				default:
					return fmt.Errorf("unexpected argument: %s (usage: --repo <path> [--task <name>] [--env KEY=VALUE]...)", args[i])
				}
			}
			if strings.TrimSpace(repoRoot) == "" {
				return fmt.Errorf("--repo is required")
			}
			cmd := exec.Command("task", "-d", repoRoot, taskName)
			cmd.Env = os.Environ()
			for k, v := range envPairs {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
			out, runErr := cmd.CombinedOutput()
			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				payload := map[string]any{
					"command":   "{{.Name}}",
					"preset":    "task-replay-emit-wrapper",
					"ok":        runErr == nil,
					"repo_root": repoRoot,
					"task":      taskName,
					"env_count": len(envPairs),
					"output":    strings.TrimSpace(string(out)),
				}
				if runErr != nil {
					payload["error"] = runErr.Error()
				}
				return enc.Encode(payload)
			}
			if runErr != nil {
				return fmt.Errorf("task wrapper failed: %w\n%s", runErr, strings.TrimSpace(string(out)))
			}
			_, err := fmt.Fprintf(os.Stdout, "{{.Name}} preset=task-replay-emit-wrapper: repo=%s task=%s env=%d\n", repoRoot, taskName, len(envPairs))
			if err == nil && len(out) > 0 {
				_, err = fmt.Fprint(os.Stdout, string(out))
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

import "github.com/gh-xj/agentcli-go"

func Bootstrap() {
	agentcli.InitLogger()
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
