package main

import (
	"os"

	"github.com/gh-xj/agentcli-go/examples/deploy-helper-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
