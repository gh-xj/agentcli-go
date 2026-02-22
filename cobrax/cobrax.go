package cobrax

import (
	"fmt"
	"os"
	"strings"

	agentcli "github.com/gh-xj/agentcli-go"
	"github.com/spf13/cobra"
)

// CommandSpec describes a subcommand registered under the root command.
type CommandSpec struct {
	Use   string
	Short string
	Run   func(*agentcli.AppContext, []string) error
}

// RootSpec defines the root command contract and shared runtime settings.
type RootSpec struct {
	Use      string
	Short    string
	Meta     agentcli.AppMeta
	Commands []CommandSpec
}

// NewRoot builds a root command with standardized persistent flags.
func NewRoot(spec RootSpec) *cobra.Command {
	root := &cobra.Command{
		Use:          spec.Use,
		Short:        spec.Short,
		SilenceUsage: true,
	}
	root.PersistentFlags().BoolP("verbose", "v", false, "enable debug logs")
	root.PersistentFlags().String("config", "", "config file path")
	root.PersistentFlags().Bool("json", false, "emit machine-readable JSON output")
	root.PersistentFlags().Bool("no-color", false, "disable colorized output")

	for _, c := range spec.Commands {
		cmdSpec := c
		child := &cobra.Command{
			Use:   cmdSpec.Use,
			Short: cmdSpec.Short,
			RunE: func(cmd *cobra.Command, args []string) error {
				app := agentcli.NewAppContext(cmd.Context())
				app.Meta = spec.Meta

				jsonFlag, _ := cmd.Flags().GetBool("json")
				configPath, _ := cmd.Flags().GetString("config")
				noColor, _ := cmd.Flags().GetBool("no-color")
				app.Values["json"] = jsonFlag
				app.Values["config"] = configPath
				app.Values["no-color"] = noColor

				if cmdSpec.Run == nil {
					return nil
				}
				return cmdSpec.Run(app, args)
			},
		}
		root.AddCommand(child)
	}
	return root
}

// Execute runs the root command and returns a deterministic process exit code.
func Execute(spec RootSpec, args []string) int {
	root := NewRoot(spec)
	root.SetArgs(args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return resolveCode(err)
	}
	return agentcli.ExitSuccess
}

func resolveCode(err error) int {
	if err == nil {
		return agentcli.ExitSuccess
	}
	if code := usageErrorCode(err); code != 0 {
		return code
	}
	return agentcli.ResolveExitCode(err)
}

func usageErrorCode(err error) int {
	if err == nil {
		return 0
	}
	text := err.Error()
	usageIndicators := []string{
		"unknown command",
		"unknown flag",
		"accepts",
		"requires",
		"usage:",
	}
	for _, marker := range usageIndicators {
		if strings.Contains(strings.ToLower(text), marker) {
			return agentcli.ExitUsage
		}
	}
	return 0
}
