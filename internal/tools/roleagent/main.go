package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type finding struct {
	Code string `json:"code"`
}

type roleContext struct {
	Findings []finding `json:"findings"`
}

type plannerOutput struct {
	SchemaVersion string   `json:"schema_version"`
	Summary       string   `json:"summary"`
	FixTargets    []string `json:"fix_targets"`
}

type fixerOutput struct {
	SchemaVersion string   `json:"schema_version"`
	Applied       []string `json:"applied"`
	Notes         string   `json:"notes"`
}

type judgerOutput struct {
	SchemaVersion string `json:"schema_version"`
	ExtraFindings []any  `json:"extra_findings"`
	Notes         string `json:"notes"`
}

func main() {
	role := flag.String("role", "", "planner|fixer|judger")
	contextPath := flag.String("context", "", "path to role context json")
	repoRoot := flag.String("repo-root", ".", "repo root for fixer actions")
	flag.Parse()

	if *role == "" || *contextPath == "" {
		fmt.Fprintln(os.Stderr, "role and context are required")
		os.Exit(2)
	}

	ctx, err := loadContext(*contextPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	switch *role {
	case "planner":
		out := plannerOutput{SchemaVersion: "v1", Summary: "deterministic planner analyzed findings", FixTargets: findingCodes(ctx.Findings)}
		_ = json.NewEncoder(os.Stdout).Encode(out)
	case "fixer":
		applied := []string{}
		if hasFinding(ctx.Findings, "generated_go_not_formatted") {
			cmd := exec.Command("zsh", "-lc", "gofmt -w scaffold.go")
			cmd.Dir = *repoRoot
			if out, err := cmd.CombinedOutput(); err == nil {
				applied = append(applied, "external: gofmt scaffold.go")
			} else {
				fmt.Fprintf(os.Stderr, "external fixer gofmt failed: %s\n", string(out))
			}
		}
		out := fixerOutput{SchemaVersion: "v1", Applied: applied, Notes: "deterministic fixer applied targeted remediations"}
		_ = json.NewEncoder(os.Stdout).Encode(out)
	case "judger":
		out := judgerOutput{SchemaVersion: "v1", ExtraFindings: []any{}, Notes: "independent judger reviewed post-fix outputs"}
		_ = json.NewEncoder(os.Stdout).Encode(out)
	default:
		fmt.Fprintln(os.Stderr, "unknown role")
		os.Exit(2)
	}
}

func loadContext(path string) (roleContext, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return roleContext{}, fmt.Errorf("read context: %w", err)
	}
	var ctx roleContext
	if err := json.Unmarshal(raw, &ctx); err != nil {
		return roleContext{}, fmt.Errorf("parse context: %w", err)
	}
	return ctx, nil
}

func hasFinding(findings []finding, code string) bool {
	for _, f := range findings {
		if strings.EqualFold(f.Code, code) {
			return true
		}
	}
	return false
}

func findingCodes(findings []finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		if strings.TrimSpace(f.Code) != "" {
			out = append(out, f.Code)
		}
	}
	if len(out) == 0 {
		out = append(out, "none")
	}
	return out
}
