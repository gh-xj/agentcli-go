package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gh-xj/agentcli-go/dal"
	"github.com/gh-xj/agentcli-go/operator"
)

// ScaffoldNewOptions controls scaffold generation behaviour.
type ScaffoldNewOptions struct {
	InExistingModule bool
	Minimal          bool
}

// ScaffoldService handles project scaffolding using injected dependencies.
type ScaffoldService struct {
	tpl  operator.TemplateOperator
	comp operator.ComplianceOperator
	fs   dal.FileSystem
	exec dal.Executor
}

// NewScaffoldService returns a new ScaffoldService.
func NewScaffoldService(
	tpl operator.TemplateOperator,
	comp operator.ComplianceOperator,
	fs dal.FileSystem,
	exec dal.Executor,
) *ScaffoldService {
	return &ScaffoldService{tpl: tpl, comp: comp, fs: fs, exec: exec}
}

// New creates a new CLI project using the golden agentcli layout.
func (s *ScaffoldService) New(baseDir, name, module string, opts ScaffoldNewOptions) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("project name is required")
	}
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "."
	}
	module = strings.TrimSpace(module)
	writeGoMod := true

	root := filepath.Join(baseDir, name)
	if err := s.ensureEmptyDir(root); err != nil {
		return "", err
	}

	if opts.InExistingModule {
		if module != "" {
			return "", errors.New("--module cannot be used with --in-existing-module")
		}
		modulePath, moduleRoot, err := s.tpl.ResolveParentModule(root)
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
		"main.go":     mainTpl,
		"cmd/root.go": rootCmdTpl,
	}
	if opts.Minimal {
		files["README.md"] = minimalReadmeTpl
	} else {
		files["internal/app/bootstrap.go"] = appBootstrapTpl
		files["internal/app/lifecycle.go"] = appLifecycleTpl
		files["internal/app/errors.go"] = appErrorsTpl
		files["internal/config/schema.go"] = configSchemaTpl
		files["internal/config/load.go"] = configLoadTpl
		files["internal/io/output.go"] = outputTpl
		files["internal/tools/smokecheck/main.go"] = smokeCheckTpl
		files["pkg/version/version.go"] = versionTpl
		files["test/e2e/cli_test.go"] = e2eTestTpl
		files["test/smoke/version.schema.json"] = smokeSchemaTpl
		files["Taskfile.yml"] = taskfileTpl
		files["README.md"] = readmeTpl

		// DAG scaffold files
		files["service/container.go"] = scaffoldContainerTpl
		files["service/wire.go"] = scaffoldWireTpl
		files["dal/interfaces.go"] = scaffoldDALInterfacesTpl
		files["dal/filesystem.go"] = scaffoldDALFilesystemTpl
		files["operator/interfaces.go"] = scaffoldOperatorInterfacesTpl
		files["operator/example_op.go"] = scaffoldOperatorExampleTpl
	}
	if writeGoMod {
		files["go.mod"] = goModTpl
	}

	cliName := filepath.Base(strings.TrimSpace(name))
	if cliName == "" || cliName == "." || cliName == string(filepath.Separator) {
		return "", errors.New("project name is required")
	}

	d := operator.TemplateData{
		Module:           module,
		Name:             cliName,
		GokitReplaceLine: s.tpl.DetectLocalReplaceLine(),
	}
	for path, body := range files {
		if err := s.tpl.RenderTemplate(filepath.Join(root, path), body, d); err != nil {
			return "", err
		}
	}
	if writeGoMod {
		if _, err := s.exec.RunInDir(root, "go", "mod", "tidy"); err != nil {
			return "", fmt.Errorf("run go mod tidy: %w", err)
		}
	}
	return root, nil
}

// AddCommand creates a command file and wires it into cmd/root.go.
func (s *ScaffoldService) AddCommand(rootDir, commandName, description, preset string) error {
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "."
	}
	if err := s.comp.ValidateCommandName(commandName); err != nil {
		return err
	}

	description = strings.TrimSpace(description)
	preset = strings.TrimSpace(preset)
	if preset != "" {
		presetDescription, ok := commandPresets[preset]
		if !ok {
			return fmt.Errorf("invalid preset %q: valid presets are %s", preset, strings.Join(SortedPresetNames(), ", "))
		}
		if description == "" {
			description = presetDescription
		}
	}
	if description == "" {
		description = fmt.Sprintf("describe %s", commandName)
	}

	cmdFile := filepath.Join(rootDir, "cmd", commandName+".go")
	if s.fs.Exists(cmdFile) {
		return fmt.Errorf("command file already exists: %s", cmdFile)
	}

	funcName := s.tpl.KebabToCamel(commandName)
	if err := s.tpl.RenderTemplate(cmdFile, addCommandTpl, operator.TemplateData{
		Name:        commandName,
		Module:      funcName,
		Description: description,
		Preset:      preset,
	}); err != nil {
		return err
	}

	rootFile := filepath.Join(rootDir, "cmd", "root.go")
	content, err := s.fs.ReadFile(rootFile)
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
	return s.fs.WriteFile(rootFile, []byte(updated), 0644)
}

// SortedPresetNames returns preset names in sorted order.
func SortedPresetNames() []string {
	names := make([]string, 0, len(commandPresets))
	for name := range commandPresets {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// PresetDescription returns the description for a preset name.
func PresetDescription(name string) (string, bool) {
	description, ok := commandPresets[name]
	return description, ok
}

func (s *ScaffoldService) ensureEmptyDir(root string) error {
	if s.fs.Exists(root) {
		entries, err := s.fs.ReadDir(root)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("target directory is not empty: %s", root)
		}
		return nil
	}
	return s.fs.EnsureDir(root)
}
