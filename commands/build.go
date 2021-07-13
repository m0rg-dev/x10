package commands

import (
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/plumbing"
	"m0rg.dev/x10/x10_log"
	"m0rg.dev/x10/x10_util"
)

type BuildCommand struct{}

func init() {
	RegisterCommand(BuildCommand{}, "build",
		"[build options] --target-root=<target root> <package name>")

	conf.RegisterKey("build", "target-root", conf.ConfigKey{
		HelpText:   "Directory to chroot (or equivalent) to during build.",
		Default:    "",
		TakesValue: true,
	})

	conf.RegisterKey("build", "reset", conf.ConfigKey{
		HelpText:   "Disable removal of auto-installed dependencies.",
		Default:    "true",
		TakesValue: false,
	})
}

func (cmd BuildCommand) Run(args []string) error {
	logger := x10_log.Get("main")

	conf.AssertArgumentCount("build", 1, args)
	conf.AssertConfigured("build", "build:target-root")
	conf.AssertConfigured("build", "repo")

	pkgdb := db.PackageDatabase{BackingFile: x10_util.PkgDb(conf.Get("build:target-root"))}
	err := pkgdb.IndexFromRepo()
	if err != nil {
		logger.Fatal(err)
	}

	if conf.GetBool("build:reset") {
		logger.Info("Removing autodeps")
		err := plumbing.Reset(logger, conf.Get("build:target-root"))
		if err != nil {
			logger.Fatal(err)
		}
	}

	return plumbing.Build(args[0])
}
