package db

import (
	"errors"

	"m0rg.dev/x10/pkgset"
	"m0rg.dev/x10/x10_log"
)

type PackageOperationType int
type PackageOperation struct {
	Fqn string
	Op  PackageOperationType
}

const (
	ActionInstall = iota
	ActionRemove
)

func (pkgdb *PackageDatabase) Plan(root string, target *pkgset.PackageSet) ([]PackageOperation, error) {
	logger := x10_log.Get("plan")

	outstanding := map[string]bool{}

	for _, pkg := range target.List() {
		outstanding[pkg] = true
	}

	pkgs, complete, err := pkgdb.Resolve(logger, outstanding)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, errors.New("package list is not complete - build first")
	}

	installed, err := pkgset.Set("installed", root)
	if err != nil {
		return nil, err
	}

	target_installed := pkgset.Empty()

	rc := []PackageOperation{}

	for _, pkg := range pkgs {
		if !installed.Check(pkg.GetFQN()) {
			logger.Debugf(" => %s", pkg.GetFQN())
			rc = append(rc, PackageOperation{pkg.GetFQN(), ActionInstall})
		}
		target_installed.Mark(pkg.GetFQN())
	}

	for _, fqn := range installed.List() {
		if !target_installed.Check(fqn) {
			logger.Debugf(" <= %s", fqn)
			rc = append(rc, PackageOperation{fqn, ActionRemove})
		}
	}

	return rc, nil
}
