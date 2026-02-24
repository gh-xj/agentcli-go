package loopapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
)

type RunRequest struct {
	Action           string  `json:"action"`
	RepoRoot         string  `json:"repo_root"`
	Threshold        float64 `json:"threshold"`
	MaxIterations    int     `json:"max_iterations"`
	Branch           string  `json:"branch"`
	Mode             string  `json:"mode"`
	RoleConfig       string  `json:"role_config"`
	VerboseArtifacts bool    `json:"verbose_artifacts"`
	Seed             int64   `json:"seed"`
	Budget           int     `json:"budget"`
}

func Serve(addr, defaultRepoRoot string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/loop/run", loopRunHandler(defaultRepoRoot))
	return http.ListenAndServe(addr, mux)
}

func loopRunHandler(defaultRepoRoot string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req RunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		repoRoot, err := resolveLoopRepoRoot(req.RepoRoot, defaultRepoRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		roleConfigPath, err := resolveLoopRoleConfig(req.RoleConfig, repoRoot)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cfg := harnessloop.Config{
			RepoRoot:         repoRoot,
			Threshold:        req.Threshold,
			MaxIterations:    req.MaxIterations,
			Branch:           req.Branch,
			Mode:             req.Mode,
			RoleConfigPath:   roleConfigPath,
			VerboseArtifacts: req.VerboseArtifacts,
			Seed:             req.Seed,
			Budget:           req.Budget,
		}
		switch req.Action {
		case "run", "judge":
			cfg.AutoFix = false
			cfg.AutoCommit = false
		case "autofix":
			cfg.AutoFix = true
			cfg.AutoCommit = true
		case "all":
			http.Error(w, "action 'all' removed; use 'autofix'", http.StatusBadRequest)
			return
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		result, err := harnessloop.RunLoop(cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			http.Error(w, "encode response failed", http.StatusInternalServerError)
			return
		}
	}
}

func resolveLoopRepoRoot(reqRepoRoot, defaultRepoRoot string) (string, error) {
	baseRoot := strings.TrimSpace(defaultRepoRoot)
	if baseRoot == "" {
		baseRoot = "."
	}
	absBase, err := filepath.Abs(baseRoot)
	if err != nil {
		return "", fmt.Errorf("resolve server repo_root: %w", err)
	}

	repoRoot := strings.TrimSpace(reqRepoRoot)
	if repoRoot == "" {
		return absBase, nil
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve request repo_root: %w", err)
	}
	if !pathWithinRoot(absRepo, absBase) {
		return "", fmt.Errorf("repo_root must stay under server root")
	}
	stat, err := os.Stat(absRepo)
	if err != nil || !stat.IsDir() {
		return "", fmt.Errorf("repo_root must be an existing directory under server root")
	}
	return absRepo, nil
}

func resolveLoopRoleConfig(roleConfig, repoRoot string) (string, error) {
	if roleConfig == "" {
		return "", nil
	}
	repoRoot = filepath.Clean(repoRoot)
	candidate := strings.TrimSpace(roleConfig)
	if candidate == "" {
		return "", nil
	}
	if filepath.IsAbs(candidate) {
		candidate = filepath.Clean(candidate)
		if !pathWithinRoot(candidate, repoRoot) {
			return "", fmt.Errorf("role_config path must be under repo_root")
		}
		return candidate, nil
	}
	resolved := filepath.Clean(filepath.Join(repoRoot, candidate))
	if !pathWithinRoot(resolved, repoRoot) {
		return "", fmt.Errorf("role_config path must be under repo_root")
	}
	return resolved, nil
}

func pathWithinRoot(path, root string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if !pathWithinLexicalRoot(path, root) {
		return false
	}

	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	resolvedPath, err := resolvePathWithSymlinkPrefix(path)
	if err != nil {
		return false
	}
	return pathWithinLexicalRoot(resolvedPath, resolvedRoot)
}

func resolvePathWithSymlinkPrefix(path string) (string, error) {
	missing := make([]string, 0, 4)
	current := filepath.Clean(path)
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func pathWithinLexicalRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
