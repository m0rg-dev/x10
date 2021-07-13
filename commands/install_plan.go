package commands

import (
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/plumbing"
	"m0rg.dev/x10/x10_log"
)

type InstallPlanCommand struct{}

func init() {
	RegisterCommand(InstallPlanCommand{}, "install_plan",
		"[install_plan options] <package name> <target>")
	conf.RegisterKey("install_plan", "dot", conf.ConfigKey{
		HelpText:   "Output dependency tree in GraphViz .dot format",
		TakesValue: false,
		Default:    "false",
	})
}

func (cmd InstallPlanCommand) Run(args []string) error {
	logger := x10_log.Get("main")

	conf.AssertArgumentCount("install_plan", 2, args)

	atom := args[0]
	target := args[1]

	pkgdb := db.PackageDatabase{BackingFile: conf.PkgDb()}
	world, err := plumbing.AddPackageToLocalWorld(pkgdb, target, atom)
	if err != nil {
		return err
	}
	plumbing.CheckPlan(logger, pkgdb, target, world)

	/*
		if (installPlanDot != nil) && *installPlanDot {
			fmt.Println("digraph {")
			fmt.Println("  rankdir = TB;")
			for idx, pkg := range pkgs {
				fmt.Printf("  \"%s\" [label=\"%d\\n%s\" shape=box];\n", pkg.GetFQN(), idx, pkg.Meta.Name)
				seen := map[string]bool{}
				for _, depend := range append(pkg.Depends.Run, pkg.GeneratedDepends...) {
					fqn := contents.ProviderIndex[depend]
					if !seen[fqn] {
						fmt.Printf("  \"%s\" -> \"%s\"\n", pkg.GetFQN(), fqn)
					}
					seen[fqn] = true
				}
			}
			fmt.Println("}")
		}
	*/
	return nil
}
