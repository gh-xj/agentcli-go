package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GenerateOptions struct {
	Mode       MigrationMode
	OutputRoot string
	Apply      bool
}

type GenerateResult struct {
	OutputRoot        string   `json:"output_root"`
	GeneratedCommands int      `json:"generated_commands"`
	GeneratedFiles    []string `json:"generated_files,omitempty"`
}

func Generate(plan Plan, opts GenerateOptions) (GenerateResult, error) {
	outputRoot, err := resolveOutputRoot(plan.RepoRoot, opts.Mode, opts.OutputRoot)
	if err != nil {
		return GenerateResult{}, err
	}
	result := GenerateResult{
		OutputRoot:     outputRoot,
		GeneratedFiles: make([]string, 0, len(plan.Scripts)+4),
	}
	if !opts.Apply {
		return result, nil
	}

	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return GenerateResult{}, fmt.Errorf("create output root: %w", err)
	}

	wrapperFiles, wrapperCount, err := generateCommandWrappers(outputRoot, plan)
	if err != nil {
		return GenerateResult{}, err
	}
	result.GeneratedFiles = append(result.GeneratedFiles, wrapperFiles...)
	result.GeneratedCommands = wrapperCount

	artifactFiles, err := writeMigrationArtifacts(outputRoot, plan)
	if err != nil {
		return GenerateResult{}, err
	}
	result.GeneratedFiles = append(result.GeneratedFiles, artifactFiles...)
	return result, nil
}

func resolveOutputRoot(repoRoot string, mode MigrationMode, outputRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("repo root is required")
	}
	mode = MigrationMode(strings.TrimSpace(string(mode)))
	if mode == "" {
		mode = ModeSafe
	}
	if mode != ModeSafe && mode != ModeInPlace {
		return "", fmt.Errorf("invalid mode: %s", mode)
	}

	root := strings.TrimSpace(outputRoot)
	if root == "" {
		if mode == ModeSafe {
			root = filepath.Join(repoRoot, "agentcli-migrated")
		} else {
			root = repoRoot
		}
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve output root: %w", err)
	}
	return absRoot, nil
}

func generateCommandWrappers(outputRoot string, plan Plan) ([]string, int, error) {
	created := make([]string, 0, len(plan.Scripts))
	count := 0
	for _, script := range plan.Scripts {
		if script.Strategy == StrategyManual {
			continue
		}
		commandName := normalizeCommandName(script.Script.Name)
		if commandName == "" {
			continue
		}

		cmdDir := filepath.Join(outputRoot, "cmd")
		if err := os.MkdirAll(cmdDir, 0o755); err != nil {
			return nil, 0, fmt.Errorf("create cmd dir: %w", err)
		}
		target := filepath.Join(cmdDir, commandName+".go")
		relScript := filepath.ToSlash(script.Script.Path)
		content := fmt.Sprintf(`package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

func %sWrapper(args []string) error {
	cmd := exec.Command("sh", append([]string{%q}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run script: %%w", err)
	}
	return nil
}
`, toExported(commandName), relScript)

		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return nil, 0, fmt.Errorf("write command wrapper: %w", err)
		}
		created = append(created, target)
		count++
	}
	return created, count, nil
}

func writeMigrationArtifacts(outputRoot string, plan Plan) ([]string, error) {
	created := make([]string, 0, 5)
	migrationDir := filepath.Join(outputRoot, "docs", "migration")
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		return nil, fmt.Errorf("create migration docs dir: %w", err)
	}

	planJSONPath := filepath.Join(migrationDir, "plan.json")
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal migration plan: %w", err)
	}
	if err := os.WriteFile(planJSONPath, planJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write plan json: %w", err)
	}
	created = append(created, planJSONPath)

	reportJSONPath := filepath.Join(migrationDir, "report.json")
	reportPayload := Report{
		SchemaVersion: "v1",
		RepoRoot:      plan.RepoRoot,
		Source:        plan.Source,
		Summary:       plan.Summary,
		Scripts:       plan.Scripts,
	}
	reportJSON, err := json.MarshalIndent(reportPayload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal migration report: %w", err)
	}
	if err := os.WriteFile(reportJSONPath, reportJSON, 0o644); err != nil {
		return nil, fmt.Errorf("write report json: %w", err)
	}
	created = append(created, reportJSONPath)

	reportPath := filepath.Join(migrationDir, "report.md")
	reportMarkdown := buildReportMarkdown(plan)
	if err := os.WriteFile(reportPath, []byte(reportMarkdown), 0o644); err != nil {
		return nil, fmt.Errorf("write report: %w", err)
	}
	created = append(created, reportPath)

	compatPath := filepath.Join(migrationDir, "compatibility.md")
	compat := buildCompatibilityMarkdown(plan)
	if err := os.WriteFile(compatPath, []byte(compat), 0o644); err != nil {
		return nil, fmt.Errorf("write compatibility report: %w", err)
	}
	created = append(created, compatPath)

	skillPath := filepath.Join(outputRoot, "skill.migration.md")
	skill := buildMigrationSkillMarkdown(plan)
	if err := os.WriteFile(skillPath, []byte(skill), 0o644); err != nil {
		return nil, fmt.Errorf("write migration skill: %w", err)
	}
	created = append(created, skillPath)

	return created, nil
}

func buildReportMarkdown(plan Plan) string {
	var b strings.Builder
	b.WriteString("# Migration Report\n\n")
	b.WriteString(fmt.Sprintf("- source: `%s`\n", plan.Source))
	b.WriteString(fmt.Sprintf("- scripts: %d\n", plan.Summary.Total))
	b.WriteString(fmt.Sprintf("- auto: %d\n", plan.Summary.Auto))
	b.WriteString(fmt.Sprintf("- wrapper: %d\n", plan.Summary.Wrapper))
	b.WriteString(fmt.Sprintf("- manual: %d\n\n", plan.Summary.Manual))
	b.WriteString("## Script Decisions\n\n")
	for _, script := range plan.Scripts {
		b.WriteString(fmt.Sprintf("- `%s`: `%s`\n", script.Script.Path, script.Strategy))
	}
	return b.String()
}

func buildCompatibilityMarkdown(plan Plan) string {
	var b strings.Builder
	b.WriteString("# Migration Compatibility\n\n")
	b.WriteString("- v1 support: `bash`, `sh`\n")
	b.WriteString("- migration style: wrapper-first for safety\n\n")
	b.WriteString("## Scripts\n\n")
	for _, script := range plan.Scripts {
		b.WriteString(fmt.Sprintf("- `%s`: shell=`%s`, strategy=`%s`\n", script.Script.Path, script.Script.Shell, script.Strategy))
	}
	return b.String()
}

func buildMigrationSkillMarkdown(plan Plan) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: migration-followup\n")
	b.WriteString("description: Follow up migration tasks generated by agentcli migrate.\n")
	b.WriteString("---\n\n")
	b.WriteString("# Migration Follow-up Skill\n\n")
	b.WriteString("## Use this when\n\n")
	b.WriteString("- You want to complete manual migration items from `docs/migration/report.md`.\n")
	b.WriteString("- You want to replace wrapper commands with native Go implementations.\n\n")
	b.WriteString("## Current summary\n\n")
	b.WriteString(fmt.Sprintf("- scripts: %d\n", plan.Summary.Total))
	b.WriteString(fmt.Sprintf("- manual required: %d\n", plan.Summary.Manual))
	return b.String()
}

func normalizeCommandName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.Trim(name, "-")
	return name
}

func toExported(name string) string {
	if name == "" {
		return "Migrated"
	}
	parts := strings.Split(name, "-")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}
