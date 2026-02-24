package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var (
	argRefPattern = regexp.MustCompile(`\$(\{?[0-9@*#?-]\}?)`)
	envRefPattern = regexp.MustCompile(`\$(\{?[A-Z_][A-Z0-9_]*\}?)`)
)

var shellBuiltins = map[string]struct{}{
	".":        {},
	"[":        {},
	"break":    {},
	"cd":       {},
	"continue": {},
	"echo":     {},
	"eval":     {},
	"exec":     {},
	"exit":     {},
	"export":   {},
	"false":    {},
	"fi":       {},
	"for":      {},
	"function": {},
	"if":       {},
	"local":    {},
	"printf":   {},
	"read":     {},
	"return":   {},
	"set":      {},
	"shift":    {},
	"source":   {},
	"test":     {},
	"then":     {},
	"true":     {},
}

func ScanScripts(repoRoot, source string) (ScanResult, error) {
	root := strings.TrimSpace(repoRoot)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ScanResult{}, fmt.Errorf("resolve repo root: %w", err)
	}

	source = strings.TrimSpace(source)
	if source == "" {
		source = "scripts"
	}

	sourcePath := source
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(absRoot, sourcePath)
	}
	sourcePath = filepath.Clean(sourcePath)

	info, err := os.Stat(sourcePath)
	if err != nil {
		return ScanResult{}, fmt.Errorf("stat source: %w", err)
	}
	if !info.IsDir() {
		return ScanResult{}, fmt.Errorf("source is not a directory: %s", source)
	}

	result := ScanResult{
		RepoRoot: absRoot,
		Source:   source,
		Scripts:  make([]ScriptInfo, 0),
	}

	walkErr := filepath.WalkDir(sourcePath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		result.Scanned++

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read script %s: %w", path, err)
		}

		shell := detectShell(path, data)
		if shell != ShellSh && shell != ShellBash {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			relPath = path
		}
		result.Scripts = append(result.Scripts, ScriptInfo{
			Name:         scriptNameFromPath(path),
			Path:         filepath.ToSlash(relPath),
			Shell:        shell,
			Executable:   isExecutable(entry),
			UsesArgs:     argRefPattern.Match(data),
			UsesEnv:      envRefPattern.Match(data),
			ExternalDeps: detectExternalDependencies(string(data)),
			RiskSignals:  detectRiskSignals(string(data)),
			SizeBytes:    len(data),
		})
		return nil
	})
	if walkErr != nil {
		return ScanResult{}, fmt.Errorf("walk source: %w", walkErr)
	}

	sort.Slice(result.Scripts, func(i, j int) bool {
		return result.Scripts[i].Path < result.Scripts[j].Path
	})
	return result, nil
}

func detectShell(path string, data []byte) ShellType {
	text := string(data)
	if idx := strings.IndexByte(text, '\n'); idx > 0 {
		firstLine := strings.TrimSpace(text[:idx])
		if strings.HasPrefix(firstLine, "#!") {
			switch {
			case strings.Contains(firstLine, "bash"):
				return ShellBash
			case strings.Contains(firstLine, "/sh") || strings.Contains(firstLine, " sh"):
				return ShellSh
			}
		}
	}
	if filepath.Ext(path) == ".sh" {
		return ShellSh
	}
	return ShellUnknown
}

func isExecutable(entry os.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}

func scriptNameFromPath(path string) string {
	base := filepath.Base(path)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" {
		return base
	}
	return name
}

func detectExternalDependencies(text string) []string {
	lines := strings.Split(text, "\n")
	deps := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		cmd := fields[0]
		if strings.Contains(cmd, "=") || strings.HasPrefix(cmd, "$") {
			continue
		}
		cmd = strings.Trim(cmd, "[]")
		if cmd == "" {
			continue
		}
		if _, ok := shellBuiltins[cmd]; ok {
			continue
		}
		if strings.Contains(cmd, "/") {
			cmd = filepath.Base(cmd)
		}
		deps[cmd] = struct{}{}
	}
	out := make([]string, 0, len(deps))
	for dep := range deps {
		out = append(out, dep)
	}
	slices.Sort(out)
	return out
}

func detectRiskSignals(text string) []string {
	low := strings.ToLower(text)
	signals := make([]string, 0, 4)
	if strings.Contains(low, "eval ") || strings.Contains(low, "\neval\t") {
		signals = append(signals, "eval")
	}
	if strings.Contains(low, "source ") || strings.Contains(low, "\n. ") {
		signals = append(signals, "source")
	}
	if strings.Contains(low, "trap ") {
		signals = append(signals, "trap")
	}
	if strings.Contains(low, "<<") {
		signals = append(signals, "heredoc")
	}
	return signals
}
