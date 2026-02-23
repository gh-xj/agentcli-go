package loopapi

import (
	"encoding/json"
	"net/http"
	"strings"

	harnessloop "github.com/gh-xj/agentcli-go/internal/harnessloop"
)

type RunRequest struct {
	Action        string  `json:"action"`
	RepoRoot      string  `json:"repo_root"`
	Threshold     float64 `json:"threshold"`
	MaxIterations int     `json:"max_iterations"`
	Branch        string  `json:"branch"`
	Mode          string  `json:"mode"`
	RoleConfig    string  `json:"role_config"`
	Seed          int64   `json:"seed"`
	Budget        int     `json:"budget"`
}

func Serve(addr, defaultRepoRoot string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/loop/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req RunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		cfg := harnessloop.Config{
			RepoRoot:       strings.TrimSpace(req.RepoRoot),
			Threshold:      req.Threshold,
			MaxIterations:  req.MaxIterations,
			Branch:         req.Branch,
			Mode:           req.Mode,
			RoleConfigPath: req.RoleConfig,
			Seed:           req.Seed,
			Budget:         req.Budget,
		}
		if cfg.RepoRoot == "" {
			cfg.RepoRoot = defaultRepoRoot
		}
		switch req.Action {
		case "run", "judge":
			cfg.AutoFix = false
			cfg.AutoCommit = false
		case "autofix", "all":
			cfg.AutoFix = true
			cfg.AutoCommit = true
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
		_ = json.NewEncoder(w).Encode(result)
	})
	return http.ListenAndServe(addr, mux)
}
