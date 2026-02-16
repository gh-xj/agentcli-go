package gokit

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

// RunCommand executes a command and returns its stdout.
// Returns an error that includes stderr on failure.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// RunOsascript executes an AppleScript and returns trimmed stdout.
func RunOsascript(script string) string {
	cmd := exec.Command("osascript", "-e", script)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// Which checks if a command exists in PATH.
func Which(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// CheckDependency fatals if a required command is not in PATH.
func CheckDependency(name, installHint string) {
	if !Which(name) {
		log.Fatal().Str("cmd", name).Str("install", installHint).Msg("required dependency not found")
	}
}
