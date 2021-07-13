package commands

import (
	"os"

	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/lib"
	"m0rg.dev/x10/plumbing"
	"m0rg.dev/x10/x10_log"
)

type InstallCommand struct{}

func init() {
	RegisterCommand("install", InstallCommand{})
}

func (cmd InstallCommand) Run(args []string) error {
	logger := x10_log.Get("main")

	pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
	atom := os.Args[2]
	target := os.Args[3]

	world, err := plumbing.AddPackageToLocalWorld(pkgdb, target, atom)
	if err != nil {
		return err
	}

	plan, err := plumbing.CheckPlan(logger, pkgdb, target, world)
	if err != nil {
		return err
	}

	contents, err := pkgdb.Read()
	if err != nil {
		return err
	}

	for _, op := range plan {
		if op.Op == db.ActionInstall {
			err := lib.Install(pkgdb, contents.Packages[op.Fqn], target)
			if err != nil {
				return err
			}
		} else {
			// TODO
			logger.Fatal("don't know how to remove packages yet")
		}
	}

	return world.Write()
}
