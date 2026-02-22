package cmd

import (
	"fmt"
	"os"

	"github.com/gh-xj/agentcli-go"
)

func init() {
}

func RequestCommand() command {
	return command{
		Description: "describe request",
		Run: func(app *agentcli.AppContext, args []string) error {
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				_, err := fmt.Fprintln(os.Stdout, "{\"command\":\"request\",\"ok\":true}")
				return err
			}
			_, err := fmt.Fprintf(os.Stdout, "request executed with %d args\n", len(args))
			return err
		},
	}
}
