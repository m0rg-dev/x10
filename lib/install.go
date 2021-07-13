package lib

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/pkgset"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/trigger"
	"m0rg.dev/x10/x10_log"
)

// TODO: attempt rollback on errors

func Install(pkgdb db.PackageDatabase, pkg spec.SpecDbData, root string) error {
	logger := x10_log.Get("install").WithField("pkg", pkg.GetFQN())
	installed, err := pkgset.Set("installed", root)
	if err != nil {
		return err
	}

	if installed.Check(pkg.GetFQN()) {
		logger.Infof("Already installed: %s", pkg.GetFQN())
		return nil
	}

	logger.Infof("Installing: %s -> %s", pkg.GetFQN(), root)

	tmp_path := filepath.Join(root, "tmp", "x10", pkg.GetFQN())

	err = os.MkdirAll(tmp_path, os.ModePerm)
	if err != nil {
		return err
	}
	extract_cmd := exec.Command("tar", "xvf", filepath.Join(conf.HostDir(), "binpkgs", pkg.GetFQN()+".tar.xz"), "-C", tmp_path)
	out, err := extract_cmd.CombinedOutput()
	if err != nil {
		logger.Error(string(out))
		return err
	}

	err = filepath.WalkDir(tmp_path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target_path, err := filepath.Rel(tmp_path, path)
		if err != nil {
			return err
		}

		if d.Type().IsRegular() && !strings.ContainsRune(target_path, '/') {
			// no files under /!
			return nil
		}

		target_path, err = filepath.Abs(filepath.Join(root, target_path))
		if err != nil {
			return err
		}

		if d.IsDir() {
			err := os.Mkdir(target_path, os.ModePerm)
			if err != nil {
				if errors.Is(err, os.ErrExist) {
					logger.Debugf(" -- %s", target_path)
				} else {
					return err
				}
			} else {
				logger.Debugf(" => %s", target_path)
			}
		} else {
			copy_cmd := exec.Command("cp", "-af", path, target_path)
			out, err := copy_cmd.CombinedOutput()
			logger.Debugf(" %s => %s", path, target_path)
			if err != nil {
				logger.Error(string(out))
				return err
			}
			if copy_cmd.ProcessState.ExitCode() != 0 {
				logger.Error(string(out))
				return fmt.Errorf("%+v exited with code %d", copy_cmd.Args, copy_cmd.ProcessState.ExitCode())
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	spec, err := pkg.ToLayer()
	if err != nil {
		return err
	}

	err = trigger.RunTriggers(*spec)
	if err != nil {
		return err
	}

	installed.Mark(pkg.GetFQN())
	return installed.Write()
}

func Remove(pkgdb db.PackageDatabase, pkg spec.SpecDbData, root string) error {
	logger := x10_log.Get("remove").WithField("pkg", pkg.GetFQN())

	extract_cmd := exec.Command("tar", "tf", filepath.Join(conf.HostDir(), "binpkgs", pkg.GetFQN()+".tar.xz"))
	out, err := extract_cmd.CombinedOutput()

	if err != nil {
		return err
	}

	files := []string{}
	dirs := []string{}

	for _, line := range strings.Split(string(out), "\n") {
		abs, err := filepath.Abs(filepath.Join(root, line))
		if err != nil {
			return err
		}

		stats, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Warn(err)
				continue
			} else {
				return err
			}
		}

		if stats.IsDir() {
			dirs = append(dirs, abs)
		} else {
			files = append(files, abs)
		}
	}

	for _, file := range files {
		err = os.Remove(file)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Warn(err)
			} else {
				return err
			}
		}
	}

	for _, dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Warn(err)
			} else {
				return err
			}
		}

		if len(ents) == 0 {
			os.Remove(dir)
		}
	}

	installed, err := pkgset.Set("installed", root)
	if err != nil {
		return err
	}
	installed.Unmark(pkg.GetFQN())
	err = installed.Write()
	if err != nil {
		return err
	}
	logger.Info("Removed: " + pkg.GetFQN())

	return nil
}
