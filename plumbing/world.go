package plumbing

import (
	"m0rg.dev/x10/db"
	"m0rg.dev/x10/pkgset"
)

func AddPackageToLocalWorld(pkgdb db.PackageDatabase, root string, atom string) (*pkgset.PackageSet, error) {
	contents, err := pkgdb.Read()
	if err != nil {
		return nil, err
	}

	pkg_fqn, err := contents.FindFQN(atom)
	if err != nil {
		return nil, err
	}

	world, err := pkgset.Set("world", root)
	if err != nil {
		return nil, err
	}

	world.Mark(*pkg_fqn)
	return world, nil
}

func GetWorld(root string) (*pkgset.PackageSet, error) {
	return pkgset.Set("world", root)
}
