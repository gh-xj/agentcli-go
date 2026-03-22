package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	repoRoot := flag.String("repo-root", ".", "repository root")
	flag.Parse()

	if err := run(*repoRoot); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("doc drift check passed")
}

func run(repoRoot string) error {
	leanSig, err := loopUsage(repoRoot, "loop")
	if err != nil {
		return err
	}
	labSig, err := loopUsage(repoRoot, "loop", "lab")
	if err != nil {
		return err
	}

	targets := []string{
		filepath.Join(repoRoot, "skills", "verification-loop", "SKILL.md"),
		filepath.Join(repoRoot, "skills", "verification-loop", "README.md"),
	}

	missing := []string{}
	for _, path := range targets {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		text := string(raw)
		if !strings.Contains(text, leanSig) {
			missing = append(missing, fmt.Sprintf("%s missing '%s'", path, leanSig))
		}
		if !strings.Contains(text, labSig) {
			missing = append(missing, fmt.Sprintf("%s missing '%s'", path, labSig))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("skill docs drift detected:\n- %s", strings.Join(missing, "\n- "))
	}
	return nil
}

func loopUsage(repoRoot string, args ...string) (string, error) {
	cmdArgs := append([]string{"run", "./cmd/agentops"}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	usage, usageErr := extractUsage(out.String())
	if usageErr == nil {
		return usage, nil
	}
	if err != nil {
		return "", fmt.Errorf("run usage command: %w\n%s", err, out.String())
	}
	return "", usageErr
}

func extractUsage(output string) (string, error) {
	usageRe := regexp.MustCompile(`usage: (agentops loop[^\n]+)`)
	match := usageRe.FindStringSubmatch(output)
	if len(match) < 2 {
		return "", fmt.Errorf("could not extract loop command signature from CLI output")
	}
	return strings.TrimSpace(match[1]), nil
}
