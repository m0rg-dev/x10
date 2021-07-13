package lib

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/runner"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

func RunStage(pkg spec.SpecLayer, stage string) error {
	logger := x10_log.Get("run").WithField("stage", stage).WithField("package", pkg.GetFQN())
	logger.Info("Running")

	if pkg.Stages[stage] == nil {
		logger.Info("  <empty stage>")
		return nil
	}
	basepath := conf.BaseDir()

	additional_args := []string{}
	logger.Debug("files dir: " + filepath.Join(basepath, "files", pkg.Meta.Name))
	_, err := os.Stat(filepath.Join(basepath, "files", pkg.Meta.Name))
	if err == nil {
		additional_args = append(additional_args, "-v")
		additional_args = append(additional_args, fmt.Sprintf("%s:%s", filepath.Join(basepath, "files", pkg.Meta.Name), "/pkgfiles"))
	}

	script_chunks := []string{}
	script_chunks = append(script_chunks, pkg.GetEnvironmentSetupScript())

	if *pkg.Stages[stage].UseWorkdir {
		script_chunks = append(script_chunks, "cd \"$X10_WORKDIR\"")
	}

	script_chunks = append(script_chunks, pkg.Stages[stage].PreScript...)
	if pkg.Stages[stage].Script != nil {
		script_chunks = append(script_chunks, *pkg.Stages[stage].Script)
	}
	script_chunks = append(script_chunks, pkg.Stages[stage].PostScript...)

	err = runner.RunTargetScript(logger, strings.Join(script_chunks, "\n"), additional_args)

	if err != nil {
		return err
	}

	if stage == "package" {
		d, err := yaml.Marshal(pkg.Meta)
		if err != nil {
			logger.Error("Error while marshalling package metadata: ")
			logger.Error(err)
			return err
		}
		err = ioutil.WriteFile(filepath.Join(conf.TargetDir(), "destdir", pkg.GetFQN(), "meta.yml"), d, fs.ModePerm)
		if err != nil {
			logger.Error("Error while writing package metadata: ")
			logger.Error(err)
			return err
		}

		d, err = yaml.Marshal(pkg.Depends)
		if err != nil {
			logger.Error("Error while marshalling package dependencies: ")
			logger.Error(err)
			return err
		}
		err = ioutil.WriteFile(filepath.Join(conf.TargetDir(), "destdir", pkg.GetFQN(), "depends.yml"), d, fs.ModePerm)
		if err != nil {
			logger.Error("Error while writing package dependencies: ")
			logger.Error(err)
			return err
		}

		db := db.PackageDatabase{BackingFile: conf.PkgDb()}
		err = db.Update(pkg, false)
		if err != nil {
			logger.Error("Error while updating package database: ")
			logger.Error(err)
			return err
		}
	}

	return nil
}
