package plumbing

import (
	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/pkgset"
)

func CheckPlan(logger *logrus.Entry, pkgdb db.PackageDatabase, root string, target *pkgset.PackageSet) ([]db.PackageOperation, error) {
	plan, err := pkgdb.Plan(root, target)
	if err != nil {
		return nil, err
	}
	if len(plan) > 0 {
		logger.Info("Here's the plan:")
		logger.Info("")

		for _, item := range plan {
			if item.Op == db.ActionInstall {
				logger.Infof(" Install:  %s", item.Fqn)
			} else {
				logger.Infof("  Remove:  %s", item.Fqn)
			}
		}
	} else {
		logger.Info("(nothing to do)")
	}
	return plan, nil
}
