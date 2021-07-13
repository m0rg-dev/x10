package db

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gofrs/flock"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"m0rg.dev/x10/conf"
	"m0rg.dev/x10/spec"
	"m0rg.dev/x10/x10_log"
)

type PackageDatabaseContents struct {
	Packages      map[string]spec.SpecDbData // FQN -> data
	ProviderIndex map[string]string          // atom -> FQN
}

type PackageDatabase struct {
	BackingFile string
}

func (db *PackageDatabase) unlocked_Read() (*PackageDatabaseContents, error) {
	raw_contents, err := ioutil.ReadFile(db.BackingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &PackageDatabaseContents{
				Packages:      map[string]spec.SpecDbData{},
				ProviderIndex: map[string]string{},
			}, nil
		}
		return nil, err
	}

	contents := &PackageDatabaseContents{}
	err = yaml.UnmarshalStrict(raw_contents, contents)
	if err != nil {
		return nil, err
	}

	if contents.Packages == nil {
		contents.Packages = map[string]spec.SpecDbData{}
	}

	if contents.ProviderIndex == nil {
		contents.ProviderIndex = map[string]string{}
	}

	return contents, nil
}

func (db *PackageDatabase) Read() (*PackageDatabaseContents, error) {
	lock := flock.New(db.BackingFile + ".lock")
	lock.RLock()
	defer lock.Close()

	contents, err := db.unlocked_Read()
	return contents, err
}

func (db *PackageDatabase) unlocked_Write(contents *PackageDatabaseContents) error {
	d, err := yaml.Marshal(contents)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(db.BackingFile, d, os.ModePerm)
}

func (db *PackageDatabase) Update(pkg spec.SpecLayer, force_invalid bool) error {
	logger := x10_log.Get("update").WithField("pkg", pkg.GetFQN())
	// Attempt to grab generated dependencies
	dbpkg := pkg.ToDB()
	dbpkg.GeneratedValid = true
	gen_depends, err := ioutil.ReadFile(filepath.Join(conf.TargetDir(), "destdir", pkg.GetFQN(), "generated-depends"))
	if err == nil {
		dbpkg.GeneratedDepends = strings.Split(strings.TrimSpace(string(gen_depends)), "\n")
	} else {
		if !os.IsNotExist(err) {
			dbpkg.GeneratedValid = false
		} else {
			logger.Debug(" => no generated depends")
		}
	}
	gen_provides, err := ioutil.ReadFile(filepath.Join(conf.TargetDir(), "destdir", pkg.GetFQN(), "generated-provides"))
	if err == nil {
		dbpkg.GeneratedProvides = strings.Split(strings.TrimSpace(string(gen_provides)), "\n")
	} else {
		if !os.IsNotExist(err) {
			dbpkg.GeneratedValid = false
		} else {
			logger.Debug(" => no generated depends")
		}
	}

	if force_invalid {
		dbpkg.GeneratedValid = false
	}

	lock := flock.New(db.BackingFile + ".lock")
	lock.Lock()
	defer lock.Close()

	contents, err := db.unlocked_Read()
	if err != nil {
		return err
	}

	contents.Packages[pkg.GetFQN()] = dbpkg
	if dbpkg.GeneratedValid {
		for _, prov := range dbpkg.GeneratedProvides {
			contents.ProviderIndex[prov] = pkg.GetFQN()
		}
	}

	// TODO only do this if package is latest
	contents.ProviderIndex[pkg.Meta.Name] = pkg.GetFQN()

	db.unlocked_Write(contents)

	logger.Info("Updated package database in " + db.BackingFile + ".")
	return err
}

func (contents *PackageDatabaseContents) CheckUpToDate(from_repo spec.SpecLayer) bool {
	logger := x10_log.Get("check").WithField("pkg", from_repo.GetFQN())

	repo_to_db := from_repo.ToDB()
	from_db, ok := contents.Packages[from_repo.GetFQN()]
	if ok {
		logger.Debugf(" => already in DB")
		if reflect.DeepEqual(repo_to_db.Meta, from_db.Meta) {
			logger.Debugf("  => meta match")
		} else {
			logger.Warnf(" => meta for %s doesn't match repo,", from_repo.GetFQN())
			logger.Debug(spew.Sdump(repo_to_db))
			logger.Debug(spew.Sdump(from_db))
			return false
		}
	} else {
		return false
	}
	return true
}

func (contents *PackageDatabaseContents) maybeAddProvider(atom string, fqn string) {
	existing, ok := contents.ProviderIndex[atom]
	if ok {
		if strings.Compare(existing, atom) > 0 {
			contents.ProviderIndex[atom] = fqn
		}
	} else {
		contents.ProviderIndex[atom] = fqn
	}
}

type DependencyType int

const (
	DepHostBuild DependencyType = iota
	DepBuild
	DepTest
	DepRun
)

func (contents *PackageDatabaseContents) FindFQN(atom string) (*string, error) {
	_, is_fqn := contents.Packages[atom]
	if is_fqn {
		return &atom, nil
	}
	fqn, have_provider := contents.ProviderIndex[atom]
	if have_provider {
		return &fqn, nil
	}
	return nil, errors.New("Can't find FQN for " + atom)
}

func (db *PackageDatabase) Resolve(logger *logrus.Entry, outstanding map[string]bool) (pkgs []spec.SpecDbData, complete bool, err error) {
	complete = true

	contents, err := db.Read()
	if err != nil {
		return nil, false, err
	}

	resolved := map[string]bool{}
	resolved_order := []string{}

	for len(outstanding) > 0 {
		logger.Debugf("(iteration; %d left)", len(outstanding))
		depends := []string{}
		for depend := range outstanding {
			depends = append(depends, depend)
		}

		sort.Strings(depends)

		for _, fqn := range depends {
			logger.Debugf("Evaluating: %s", fqn)
			pkg := contents.Packages[fqn]
			all_depends := pkg.Depends.Run

			if conf.UseGeneratedDependencies() {
				if !pkg.GeneratedValid {
					logger.Warnf("Need to evaluate %s but no generated depends", fqn)
					complete = false
				}
				all_depends = append(all_depends, pkg.GeneratedDepends...)
			}

			all_resolved := true
			for _, sub_depend := range all_depends {
				depend_fqn, err := contents.FindFQN(sub_depend)
				if err != nil {
					return nil, false, err
				}
				if fqn != *depend_fqn && !resolved[*depend_fqn] {
					logger.Debugf(" => outstanding dependency: %s", *depend_fqn)
					outstanding[*depend_fqn] = true
					all_resolved = false
				}
			}

			if all_resolved {
				resolved_order = append(resolved_order, fqn)
				resolved[fqn] = true
				delete(outstanding, fqn)
				logger.Debugf(" => RESOLVED: %s", fqn)
			}
		}
	}

	seen := map[string]bool{}

	for _, fqn := range resolved_order {
		if !seen[fqn] {
			pkgs = append(pkgs, contents.Packages[fqn])
		}
		seen[fqn] = true
	}

	return pkgs, complete, nil
}

func (db *PackageDatabase) GetInstallDeps(top_level string, dep_type DependencyType) (pkgs []spec.SpecDbData, complete bool, err error) {
	logger := x10_log.Get("index").WithField("toplevel", top_level)

	contents, err := db.Read()
	if err != nil {
		return nil, false, err
	}
	outstanding := map[string]bool{}

	top_level_fqn, err := contents.FindFQN(top_level)
	if err != nil {
		return nil, false, err
	}
	top_level_pkg := contents.Packages[*top_level_fqn]

	switch dep_type {
	case DepRun:
		outstanding[*top_level_fqn] = true
	case DepTest:
		for _, atom := range top_level_pkg.Depends.Test {
			fqn, err := contents.FindFQN(atom)
			if err != nil {
				return nil, false, err
			}
			outstanding[*fqn] = true
		}
	case DepBuild:
		for _, atom := range top_level_pkg.Depends.Build {
			fqn, err := contents.FindFQN(atom)
			if err != nil {
				return nil, false, err
			}
			outstanding[*fqn] = true
		}
	case DepHostBuild:
		for _, atom := range top_level_pkg.Depends.HostBuild {
			fqn, err := contents.FindFQN(atom)
			if err != nil {
				return nil, false, err
			}
			outstanding[*fqn] = true
		}
	}

	return db.Resolve(logger, outstanding)
}

func (db *PackageDatabase) Get(fqn string) (spec.SpecDbData, error) {
	contents, err := db.Read()
	if err != nil {
		return spec.SpecDbData{}, err
	}
	return contents.Packages[fqn], nil
}
