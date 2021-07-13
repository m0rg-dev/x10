package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	"m0rg.dev/x10/commands"
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/lib"
	"m0rg.dev/x10/plumbing"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

func main() {

	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	//buildStage := buildCmd.String("stage", "", "Run only a specific stage.")
	buildMaybe := buildCmd.Bool("maybe", false, "Only build if the package is outdated in the database.")
	buildDeps := buildCmd.Bool("with_deps", false, "Build a package's runtime dependencies after building it.")

	config_path := flag.String("config", "/etc/x10.conf:./etc/x10.conf", "Colon-separated list of configuration files")
	no_reset_packages := flag.Bool("no_reset_packages", false, "")
	flag.Parse()

	conf.ReadConfig(*config_path)
	if *no_reset_packages {
		conf.Set("reset_packages", "false")
	}

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <subcommand> ...\n", os.Args[0])
		os.Exit(1)
	}

	logger := x10_log.Get("main")
	// TODO silly hack
	os.Args = append([]string{os.Args[0]}, flag.Args()...)

	switch os.Args[1] {
	case "gensum":
		pkgsrc := os.Args[2]
		pkg, err := spec.LoadPackage(pkgsrc)
		if err != nil {
			logger.Fatal(err)
		}

		lib.RunStage(*pkg, "fetch")
		lib.RunStage(*pkg, "_gensum")
	case "build":
		buildCmd.Parse(os.Args[2:])
		pkgsrc := buildCmd.Arg(0)
		pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
		contents, err := pkgdb.Read()
		if err != nil {
			logger.Fatal(err)
		}
		pkg, err := spec.LoadPackage(pkgsrc)
		if err != nil {
			logger.Fatal(err)
		}

		if (buildMaybe == nil || !*buildMaybe) ||
			!contents.CheckUpToDate(*pkg) ||
			!contents.Packages[pkg.GetFQN()].GeneratedValid {
			err = plumbing.Build(pkgdb, *pkg)
			if err != nil {
				logger.Fatal(err)
			}

			if buildDeps != nil && *buildDeps {
				complete := false
				var deps []spec.SpecDbData
				for !complete {
					var err error
					deps, complete, err = pkgdb.GetInstallDeps(pkg.GetFQN(), db.DepRun)
					if err != nil {
						logger.Fatal(err)
					}

					for _, dep := range deps {
						logger.Infof(" => depends on %s", dep.GetFQN())
						if dep.GeneratedValid {
							logger.Infof("  (already built)")
						} else {
							from_repo, err := spec.LoadPackage(filepath.Join(conf.PackageDir(), dep.Meta.Name+".yml"))
							if err != nil {
								logger.Fatal(err)
							}

							err = plumbing.Build(pkgdb, *from_repo)
							if err != nil {
								logger.Fatal(err)
							}
						}
					}
				}
			}
		}

		if conf.ResetPackages() {
			logger.Info("Removing autodeps")
			plumbing.Reset(logger, conf.TargetDir())
		}
	case "show":
		buildCmd.Parse(os.Args[2:])
		pkgsrc := buildCmd.Arg(0)
		pkg, err := spec.LoadPackage(pkgsrc)
		if err != nil {
			logger.Fatal(err)
		}
		spew.Dump(pkg)
	case "showdb":
		pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
		contents, err := pkgdb.Read()
		if err != nil {
			logger.Fatal(err)
		}
		spew.Dump(contents)
	case "index":
		pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
		err := pkgdb.IndexFromRepo()
		if err != nil {
			logger.Fatal(err)
		}
	case "list_install":
		pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
		atom := os.Args[2]

		pkgs, _, err := pkgdb.GetInstallDeps(atom, db.DepRun)
		if err != nil {
			logger.Fatal(err)
		}

		for _, pkg := range pkgs {
			fmt.Println(pkg.GetFQN())
		}
	case "list_build":
		pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
		atom := os.Args[2]

		pkgs, complete, err := pkgdb.GetInstallDeps(atom, db.DepBuild)
		if err != nil {
			logger.Fatal(err)
		}

		for _, pkg := range pkgs {
			fmt.Println(pkg.GetFQN())
		}
		if !complete {
			logger.Warn(" (package list may not be complete)")
		}
	default:
		commands.RunCommand(os.Args[1], os.Args)
	}
}
