package db

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofrs/flock"
	"golang.org/x/sync/errgroup"
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

func (db *PackageDatabase) IndexFromRepo() error {
	logger := x10_log.Get("index").WithField("db", db.BackingFile)
	lock := flock.New(db.BackingFile + ".lock")
	lock.Lock()
	defer lock.Close()

	contents, err := db.unlocked_Read()
	if err != nil {
		return err
	}

	group := new(errgroup.Group)

	var updates sync.Map

	filepath.WalkDir(conf.PackageDir(), func(path string, d fs.DirEntry, err error) error {
		if d.Name() == "layers" {
			return fs.SkipDir
		}
		if d.Type().IsRegular() {
			group.Go(func() error {
				local_logger := logger.WithField("pkgsrc", path)
				local_logger.Infof("Indexing")

				from_repo, err := spec.LoadPackage(path)
				if err != nil {
					return err
				}

				srcstat, err := os.Stat(path)
				if err != nil {
					return err
				}
				binpkg_path := filepath.Join(conf.HostDir(), "binpkgs", from_repo.GetFQN()+".tar.xz")
				pkgstat, err := os.Stat(binpkg_path)
				doupdate := false

				if !contents.CheckUpToDate(*from_repo) {
					local_logger.Infof("Updating database (outdated)")
					doupdate = true
				}

				if !doupdate && !contents.Packages[from_repo.GetFQN()].GeneratedValid {
					if err == nil {
						// TODO: did I forget to put something here?
					} else {
						local_logger.Infof("Updating database (not built)")
						doupdate = true
					}
				}

				if !doupdate && err != nil {
					local_logger.Infof("Updating database (stat error on binpkg)")
					doupdate = true
				}

				if srcstat != nil && pkgstat != nil && srcstat.ModTime().Unix() > pkgstat.ModTime().Unix() {
					local_logger.Infof("Updating database (source is newer)")
					doupdate = true
				}

				if doupdate {
					repo_to_db := from_repo.ToDB()
					updates.Store(from_repo.GetFQN(), repo_to_db)
				}
				return nil
			})
		}

		return nil
	})

	err = group.Wait()
	if err != nil {
		return err
	}

	updates.Range(func(key interface{}, value interface{}) bool {
		fqn := key.(string)
		dbpkg := value.(spec.SpecDbData)
		contents.Packages[fqn] = dbpkg
		return true
	})

	logger.Infof("Rebuilding provider cache")
	contents.ProviderIndex = map[string]string{}
	for fqn, dbpkg := range contents.Packages {
		logger.Debugf(" => " + fqn)

		local_logger := logger.WithField("fqn", fqn)

		if !dbpkg.GeneratedValid {
			ok := true
			binpkg_path := filepath.Join(conf.HostDir(), "binpkgs", dbpkg.GetFQN()+".tar.xz")
			_, err := os.Stat(binpkg_path)
			if err == nil {
				local_logger.Info("Pulling generated info from binpkg")
				generated_depends, err := getFileFromBinpkg(binpkg_path, "./generated-depends")
				if err == nil {
					dbpkg.GeneratedDepends = strings.Split(strings.TrimSpace(string(generated_depends)), "\n")
				} else {
					if !strings.Contains(generated_depends, "Not found in archive") {
						local_logger.Warn(err)
						local_logger.Warn(generated_depends)
						ok = false
					}
				}

				generated_provides, err := getFileFromBinpkg(binpkg_path, "./generated-provides")
				if err == nil {
					dbpkg.GeneratedProvides = strings.Split(strings.TrimSpace(string(generated_provides)), "\n")
				} else {
					if !strings.Contains(generated_provides, "Not found in archive") {
						local_logger.Warn(err)
						local_logger.Warn(generated_provides)
						ok = false
					}
				}

				if ok {
					dbpkg.GeneratedValid = true
					contents.Packages[dbpkg.GetFQN()] = dbpkg
				}
			}
		}

		if dbpkg.GeneratedValid {
			for _, prov := range dbpkg.GeneratedProvides {
				contents.maybeAddProvider(prov, fqn)
			}
		}
		contents.maybeAddProvider(dbpkg.Meta.Name, fqn)
	}

	db.unlocked_Write(contents)
	logger.Info("Updated package database in " + db.BackingFile + ".")
	return nil
}

func getFileFromBinpkg(binpkg_path string, file string) (string, error) {
	cmd := exec.Command("tar", "xf", binpkg_path, file, "-O")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
