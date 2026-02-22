package cmd

import (
	"fmt"
	"os"

	"github.com/gh-xj/agentcli-go"
)

func init() {
}

func DeployCommand() command {
	return command{
		Description: "describe deploy",
		Run: func(app *agentcli.AppContext, args []string) error {
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				_, err := fmt.Fprintln(os.Stdout, "{\"command\":\"deploy\",\"ok\":true}")
				return err
			}
			_, err := fmt.Fprintf(os.Stdout, "deploy executed with %d args\n", len(args))
			return err
		},
	}
}
