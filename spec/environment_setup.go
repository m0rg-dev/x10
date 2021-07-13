package spec

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (pkg SpecLayer) GetEnvironmentSetupScript() string {
	arrays := make(map[string][]string)
	vars := make(map[string]string)

	// Arrays from the package.
	for _, source := range pkg.Sources {
		arrays["X10_SOURCES_URLS"] = append(arrays["X10_SOURCES_URLS"], source.URL)
		arrays["X10_SOURCES_CHECKSUMS"] = append(arrays["X10_SOURCES_CHECKSUMS"], source.Checksum)
	}
	arrays["X10_DEPENDS_HOSTBUILDS"] = pkg.Depends.HostBuild
	arrays["X10_DEPENDS_BUILDS"] = pkg.Depends.Build
	arrays["X10_DEPENDS_TESTS"] = pkg.Depends.Test
	arrays["X10_DEPENDS_RUNS"] = pkg.Depends.Run
	if pkg.Patches != nil {
		arrays["X10_PATCHES"] = *pkg.Patches
	}

	// Package metadata.
	vars["X10_META_NAME"] = pkg.Meta.Name
	vars["X10_META_VERSION"] = pkg.Meta.Version
	vars["X10_META_REVISION"] = strconv.Itoa(pkg.Meta.Revision)
	vars["X10_META_MAINTAINER"] = pkg.Meta.Maintainer
	vars["X10_META_HOMEPAGE"] = pkg.Meta.Homepage
	vars["X10_META_LICENSE"] = pkg.Meta.License
	vars["X10_META_DESCRIPTION"] = pkg.Meta.Description
	if pkg.Meta.UnpackDir == nil {
		vars["X10_META_UNPACK_DIR"] = pkg.Meta.Name
	} else {
		vars["X10_META_UNPACK_DIR"] = *pkg.Meta.UnpackDir
	}
	vars["X10_PACKAGE_FQN"] = pkg.GetFQN()

	// System setup.
	vars["X10_MAKE_JOBS"] = strconv.Itoa(runtime.NumCPU())
	vars["DESTDIR"] = filepath.Join("/destdir", pkg.GetFQN())

	// Custom environment.
	for name, value := range pkg.Environment {
		vars[name] = value
	}

	lines := []string{}

	for name, val := range vars {
		lines = append(lines, fmt.Sprintf("export %s=%s", name, strconv.Quote(val)))
	}

	// TODO do this for everything so we have a set order
	lines = append(lines, "export X10_WORKDIR="+strconv.Quote(pkg.Workdir))

	for name, arr := range arrays {
		arr_quoted := []string{}
		for _, val := range arr {
			arr_quoted = append(arr_quoted, strconv.Quote(val))
		}
		lines = append(lines, fmt.Sprintf("export %s=(%s)", name, strings.Join(arr_quoted, " ")))
	}

	return strings.Join(lines, "\n")
}
