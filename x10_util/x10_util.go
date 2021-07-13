package x10_util

import "path/filepath"

func PkgDb(root string) string {
	return filepath.Join(root, "var", "db", "x10", "pkgdb.yml")
}
