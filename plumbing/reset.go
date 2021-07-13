package plumbing

import (
	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/lib"
)

func Reset(logger *logrus.Entry, root string) error {
	pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}

	world, err := GetWorld(root)
	if err != nil {
		return err
	}

	contents, err := pkgdb.Read()
	if err != nil {
		return err
	}
	fqn, err := contents.FindFQN("virtual/base-minimal")
	if err != nil {
		return err
	}

	world.Clear()
	world.Mark(*fqn)
	plan, err := CheckPlan(logger, pkgdb, root, world)
	if err != nil {
		return err
	}

	for _, op := range plan {
		if op.Op == db.ActionInstall {
			err := lib.Install(pkgdb, contents.Packages[op.Fqn], root)
			if err != nil {
				return err
			}
		} else {
			err := lib.Remove(pkgdb, contents.Packages[op.Fqn], root)
			if err != nil {
				return err
			}
		}
	}

	return world.Write()
}
