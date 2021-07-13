package plumbing

import (
	"path/filepath"

	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/lib"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

func Build(pkgdb db.PackageDatabase, pkg spec.SpecLayer) error {
	logger := x10_log.Get("build").WithField("pkg", pkg.GetFQN())
	complete := false
	var deps []spec.SpecDbData

	logger.Info("Finding dependencies")
	for !complete {
		local_logger := logger.WithField("type", "build")
		var err error
		var deps_2 []spec.SpecDbData
		deps_2, complete, err = pkgdb.GetInstallDeps(pkg.GetFQN(), db.DepBuild)
		deps = append(deps, deps_2...)
		if err != nil {
			return err
		}

		for _, dep := range deps_2 {
			local_logger.Infof(" => depends on %s", dep.GetFQN())
			dep, err = pkgdb.Get(dep.GetFQN())
			if err != nil {
				return err
			}

			if dep.GeneratedValid {
				local_logger.Infof("  (already built)")
			} else {
				subpkg, err := spec.LoadPackage(filepath.Join(conf.PackageDir(), dep.Meta.Name+".yml"))
				if err != nil {
					return err
				}

				err = Build(pkgdb, *subpkg)
				if err != nil {
					return err
				}
			}
		}
	}

	complete = false

	for !complete {
		local_logger := logger.WithField("type", "test")
		var err error
		var deps_2 []spec.SpecDbData
		// TODO this should all be per-stage
		deps_2, complete, err = pkgdb.GetInstallDeps(pkg.GetFQN(), db.DepTest)
		deps = append(deps, deps_2...)
		if err != nil {
			return err
		}

		for _, dep := range deps {
			local_logger.Infof(" => depends on %s", dep.GetFQN())
			dep, err = pkgdb.Get(dep.GetFQN())
			if err != nil {
				return err
			}

			if dep.GeneratedValid {
				local_logger.Infof("  (already built)")
			} else {
				subpkg, err := spec.LoadPackage(filepath.Join(conf.PackageDir(), dep.Meta.Name+".yml"))
				if err != nil {
					return err
				}

				err = Build(pkgdb, *subpkg)
				if err != nil {
					return err
				}
			}
		}
	}

	if conf.ResetPackages() {
		logger.Info("Removing autodeps")
		Reset(logger, conf.TargetDir())
	}

	for _, dep := range deps {
		err := lib.Install(pkgdb, dep, conf.TargetDir())
		if err != nil {
			return err
		}
	}
	logger.Infof("Building: %s", pkg.GetFQN())
	for _, stage := range *pkg.StageOrder {
		err := lib.RunStage(pkg, stage)
		if err != nil {
			return err
		}
	}

	return nil
}
