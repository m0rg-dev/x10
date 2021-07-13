package conf

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"m0rg.dev/x10/x10_log"
)

var config = map[string]string{}

func ReadConfig(path string) {
	logger := x10_log.Get("readconfig")
	paths := strings.Split(path, ":")
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			logger.Warn(err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			str := scanner.Text()
			split := strings.Split(str, "=")
			split[0] = strings.TrimSpace(split[0])
			split[1] = strings.TrimSpace(strings.Join(split[1:], "="))
			config[split[0]] = split[1]
		}

		if err := scanner.Err(); err != nil {
			logger.Fatal(err)
		}
	}
}

func get(key string, def string) string {
	// from_env, ok := os.LookupEnv(strings.ToUpper("x10_" + key))
	from_env, ok := config[key]
	if !ok {
		from_env = def
	}
	return from_env
}

func Set(key string, val string) {
	config[key] = val
}

func TargetDir() string {
	return get("targetdir", "./targetdir")
}

func HostDir() string {
	return get("hostdir", "./hostdir")
}

func PkgDb() string {
	rc := filepath.Join(HostDir(), "binpkgs", "pkgdb.yml")
	os.MkdirAll(filepath.Join(HostDir(), "binpkgs"), os.ModePerm)
	return rc
}

func PackageDir() string {
	return get("packagedir", "./pkgs")
}

func BaseDir() string {
	_, b, _, _ := runtime.Caller(0)
	basepath, _ := filepath.Abs(filepath.Join(filepath.Dir(b), "../.."))
	return basepath
}

func UseGeneratedDependencies() bool {
	rc := get("use_generated_dependencies", "true")
	if rc == "true" {
		return true
	} else {
		return false
	}
}

func ResetPackages() bool {
	rc := get("reset_packages", "true")
	if rc == "true" {
		return true
	} else {
		return false
	}
}
