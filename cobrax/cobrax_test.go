package cobrax

import (
	"testing"

	agentcli "github.com/gh-xj/agentcli-go"
)

func TestNewRootHasRequiredPersistentFlags(t *testing.T) {
	root := NewRoot(RootSpec{Use: "demo"})
	required := []string{"verbose", "config", "json", "no-color"}
	for _, name := range required {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("missing persistent flag: %s", name)
		}
	}
}

func TestExecuteSuccess(t *testing.T) {
	spec := RootSpec{
		Use: "demo",
		Meta: agentcli.AppMeta{
			Name: "demo",
		},
		Commands: []CommandSpec{
			{
				Use: "ping",
				Run: func(*agentcli.AppContext, []string) error {
					return nil
				},
			},
		},
	}
	code := Execute(spec, []string{"ping"})
	if code != agentcli.ExitSuccess {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestExecuteUsageErrorForUnknownCommand(t *testing.T) {
	spec := RootSpec{
		Use: "demo",
		Commands: []CommandSpec{
			{
				Use: "ping",
				Run: func(*agentcli.AppContext, []string) error {
					return nil
				},
			},
		},
	}
	code := Execute(spec, []string{"unknown"})
	if code != agentcli.ExitUsage {
		t.Fatalf("expected usage exit code, got %d", code)
	}
}

func TestExecuteTypedExitCode(t *testing.T) {
	spec := RootSpec{
		Use: "demo",
		Commands: []CommandSpec{
			{
				Use: "fail",
				Run: func(*agentcli.AppContext, []string) error {
					return agentcli.NewCLIError(agentcli.ExitPreflightDependency, "preflight", "missing dependency", nil)
				},
			},
		},
	}
	code := Execute(spec, []string{"fail"})
	if code != agentcli.ExitPreflightDependency {
		t.Fatalf("expected %d, got %d", agentcli.ExitPreflightDependency, code)
	}
}
