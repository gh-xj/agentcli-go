package cmd

import (
	"fmt"
	"os"

	"github.com/gh-xj/agentcli-go"
)

func init() {
}

func SyncCommand() command {
	return command{
		Description: "describe sync",
		Run: func(app *agentcli.AppContext, args []string) error {
			if jsonOutput, _ := app.Values["json"].(bool); jsonOutput {
				_, err := fmt.Fprintln(os.Stdout, "{\"command\":\"sync\",\"ok\":true}")
				return err
			}
			_, err := fmt.Fprintf(os.Stdout, "sync executed with %d args\n", len(args))
			return err
		},
	}
}
