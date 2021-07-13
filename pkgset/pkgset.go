package pkgset

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v2"
)

type PackageSet struct {
	backing_file string
	contents     map[string]bool
}

func (set *PackageSet) Read() error {
	lock := flock.New(set.backing_file + ".lock")
	lock.RLock()
	defer lock.Close()

	raw_contents, err := ioutil.ReadFile(set.backing_file)
	if err != nil {
		if os.IsNotExist(err) {
			set.contents = map[string]bool{}
			return nil
		}
		return err
	}

	contents := map[string]bool{}
	err = yaml.UnmarshalStrict(raw_contents, &contents)
	if err != nil {
		return err
	}

	set.contents = contents

	return nil
}

func (set *PackageSet) Write() error {
	lock := flock.New(set.backing_file + ".lock")
	lock.Lock()
	defer lock.Close()

	os.MkdirAll(filepath.Dir(set.backing_file), os.ModePerm)

	d, err := yaml.Marshal(set.contents)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(set.backing_file, d, os.ModePerm)
}

func (set *PackageSet) Mark(fqn string) {
	set.contents[fqn] = true
}

func (set *PackageSet) Check(fqn string) bool {
	return set.contents[fqn]
}

func (set *PackageSet) List() []string {
	keys := make([]string, len(set.contents))
	i := 0
	for key := range set.contents {
		keys[i] = key
		i++
	}
	return keys
}

func (set *PackageSet) Unmark(fqn string) {
	delete(set.contents, fqn)
}

func (set *PackageSet) Clear() {
	set.contents = map[string]bool{}
}

func Set(name string, root string) (*PackageSet, error) {
	r := PackageSet{filepath.Join(root, "var", "db", "x10", name), nil}
	err := r.Read()

	return &r, err
}

func Empty() *PackageSet {
	return &PackageSet{"/dev/null", map[string]bool{}}
}
