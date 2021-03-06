package commands

import (
	"fmt"
	"os"

	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/x10_log"
)

type Command interface {
	Run(args []string) error
}

var registry = map[string]Command{}

func RegisterCommand(cmd Command, name string, synopsis string) {
	registry[name] = cmd
	conf.RegisterCommand(name, synopsis)
}

func RunCommand(name string, args []string) {
	cmd, ok := registry[name]
	if ok {
		err := cmd.Run(args)
		if err != nil {
			x10_log.Get(name).Fatal(err)
		}
	} else {
		fmt.Printf("Usage: %s <subcommand> ...\n", os.Args[0])
		os.Exit(1)
	}
}
