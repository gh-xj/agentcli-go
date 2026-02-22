package main

import (
	"os"

	"github.com/gh-xj/agentcli-go/examples/http-client-cli/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:]))
}
