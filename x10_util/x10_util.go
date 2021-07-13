package x10_util

import (
	"path/filepath"

	"m0rg.dev/x10/conf"
)

func PkgDb(root string) string {
	return filepath.Join(root, "var", "db", "x10", "pkgdb.yml")
}

func PkgSrc(name string) string {
	return filepath.Join(conf.Get("packages"), name+".yml")
}
