package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/gh-xj/agentcli-go"
	"github.com/gh-xj/agentcli-go/cobrax"
)

type command struct {
	Description string
	Run         func(*agentcli.AppContext, []string) error
}

var commandRegistry = map[string]command{}

func init() {
	registerBuiltins()
	registerCommand("request", RequestCommand())
	// agentcli:add-command
}

func registerCommand(name string, cmd command) {
	commandRegistry[name] = cmd
}

func registerBuiltins() {
	registerCommand("version", command{
		Description: "print build metadata",
		Run: func(app *agentcli.AppContext, _ []string) error {
			data := map[string]string{
				"schema_version": "v1",
				"name":           "http-client-cli",
				"version":        "dev",
				"commit":         "none",
				"date":           "unknown",
			}
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(data)
			}
			_, err := fmt.Fprintf(os.Stdout, "%s %s (%s %s)\n", data["name"], data["version"], data["commit"], data["date"])
			return err
		},
	})
}

func Execute(args []string) int {
	commands := make([]cobrax.CommandSpec, 0, len(commandRegistry))
	names := make([]string, 0, len(commandRegistry))
	for name := range commandRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cmd := commandRegistry[name]
		commands = append(commands, cobrax.CommandSpec{
			Use:   name,
			Short: cmd.Description,
			Run:   cmd.Run,
		})
	}

	return cobrax.Execute(cobrax.RootSpec{
		Use:   "http-client-cli",
		Short: "http-client-cli CLI",
		Meta: agentcli.AppMeta{
			Name:    "http-client-cli",
			Version: "dev",
			Commit:  "none",
			Date:    "unknown",
		},
		Commands: commands,
	}, args)
}
