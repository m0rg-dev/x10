package plumbing

import (
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/lib"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
	"m0rg.dev/x10/x10_util"
)

func Build(name string) error {
	logger := x10_log.Get("build").WithField("pkg", name)

	pkg, err := spec.LoadPackage(x10_util.PkgSrc(name))
	if err != nil {
		logger.Fatal(err)
	}

	root := conf.Get("build:target-root")
	pkgdb := db.PackageDatabase{BackingFile: x10_util.PkgDb(root)}
	contents, err := pkgdb.Read()
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Building: %s", pkg.GetFQN())
	for _, stage := range *pkg.StageOrder {
		// err := lib.RunStage(*pkg, stage)
		// if err != nil {
		// 	return err
		// }
		logger.Infof(stage)

		outstanding := map[string]bool{}
		if stage == "configure" || stage == "build" || stage == "install" {
			logger.Infof("Finding dependencies (build).")
			for _, atom := range pkg.Depends.Build {
				fqn, err := contents.FindFQN(atom)
				if err != nil {
					return err
				}
				outstanding[*fqn] = true
			}
		} else if stage == "test" {
			logger.Infof("Finding dependencies (test).")
			for _, atom := range pkg.Depends.Test {
				fqn, err := contents.FindFQN(atom)
				if err != nil {
					return err
				}
				outstanding[*fqn] = true
			}
		}

		pkgs, complete, err := pkgdb.Resolve(logger, outstanding)
		if err != nil {
			return err
		}

		if !complete {
			// TODO.
			logger.Warn("incomplete")
		}

		for _, dep := range pkgs {
			logger.Infof("To install: " + dep.GetFQN())
			if !dep.GeneratedValid {
				err = Build(dep.Meta.Name)
				if err != nil {
					return err
				}
			}
		}

		if conf.GetBool("build:reset") {
			logger.Info("Ensuring package state.")

			world, err := GetWorld(root)
			if err != nil {
				return err
			}

			fqn, err := contents.FindFQN("virtual/base-minimal")
			if err != nil {
				return err
			}

			world.Clear()
			world.Mark(*fqn)
			for _, dep := range pkgs {
				world.Mark(dep.GetFQN())
			}
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

			err = world.Write()
			if err != nil {
				return err
			}
		} else {
			for _, dep := range pkgs {
				err := lib.Install(pkgdb, dep, root)
				if err != nil {
					return err
				}
			}
		}

		err = lib.RunStage(*pkg, stage, root)
		if err != nil {
			return err
		}
	}

	logger.Infof("Building dependencies (run).")
	for _, atom := range pkg.Depends.Run {
		fqn, err := contents.FindFQN(atom)
		if err != nil {
			return err
		}
		dep, err := pkgdb.Get(*fqn)
		if err != nil {
			return err
		}

		if !dep.GeneratedValid {
			err = Build(atom)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// func _Build(pkgdb db.PackageDatabase, pkg spec.SpecLayer) error {
// 	logger := x10_log.Get("build").WithField("pkg", pkg.GetFQN())
// 	complete := false
// 	var deps []spec.SpecDbData

// 	logger.Info("Finding dependencies")
// 	for !complete {
// 		local_logger := logger.WithField("type", "build")
// 		var err error
// 		var deps_2 []spec.SpecDbData
// 		deps_2, complete, err = pkgdb.GetInstallDeps(pkg.GetFQN(), db.DepBuild)
// 		deps = append(deps, deps_2...)
// 		if err != nil {
// 			return err
// 		}

// 		for _, dep := range deps_2 {
// 			local_logger.Infof(" => depends on %s", dep.GetFQN())
// 			dep, err = pkgdb.Get(dep.GetFQN())
// 			if err != nil {
// 				return err
// 			}

// 			if dep.GeneratedValid {
// 				local_logger.Infof("  (already built)")
// 			} else {
// 				subpkg, err := spec.LoadPackage(filepath.Join(conf.Get("packages"), dep.Meta.Name+".yml"))
// 				if err != nil {
// 					return err
// 				}

// 				err = _Build(pkgdb, *subpkg)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	complete = false

// 	for !complete {
// 		local_logger := logger.WithField("type", "test")
// 		var err error
// 		var deps_2 []spec.SpecDbData
// 		// TODO this should all be per-stage
// 		deps_2, complete, err = pkgdb.GetInstallDeps(pkg.GetFQN(), db.DepTest)
// 		deps = append(deps, deps_2...)
// 		if err != nil {
// 			return err
// 		}

// 		for _, dep := range deps {
// 			local_logger.Infof(" => depends on %s", dep.GetFQN())
// 			dep, err = pkgdb.Get(dep.GetFQN())
// 			if err != nil {
// 				return err
// 			}

// 			if dep.GeneratedValid {
// 				local_logger.Infof("  (already built)")
// 			} else {
// 				subpkg, err := spec.LoadPackage(filepath.Join(conf.Get("packages"), dep.Meta.Name+".yml"))
// 				if err != nil {
// 					return err
// 				}

// 				err = _Build(pkgdb, *subpkg)
// 				if err != nil {
// 					return err
// 				}
// 			}
// 		}
// 	}

// 	if conf.GetBool("reset") {
// 		logger.Info("Removing autodeps")
// 		Reset(logger, conf.TargetDir())
// 	}

// 	for _, dep := range deps {
// 		err := lib.Install(pkgdb, dep, conf.TargetDir())
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	logger.Infof("Building: %s", pkg.GetFQN())
// 	for _, stage := range *pkg.StageOrder {
// 		err := lib.RunStage(pkg, stage)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }
