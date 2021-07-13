package commands

import (
	"os"

	"m0rg.dev/x10/plumbing"
	"m0rg.dev/x10/x10_log"
)

type ResetCommand struct{}

func init() {
	RegisterCommand(ResetCommand{}, "reset", "Uninstall all packages except for base-minimal.")
}

func (cmd ResetCommand) Run(args []string) error {
	logger := x10_log.Get("main")

	target := os.Args[2]
	return plumbing.Reset(logger, target)
}
