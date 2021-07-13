package runner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
	"m0rg.dev/x10/conf"
)

func RunTargetScript(logger *logrus.Entry, script string, additional_podman_args []string) (err error) {
	basepath := conf.BaseDir()

	hostdir, err := filepath.Abs(conf.HostDir())
	if err != nil {
		return err
	}

	targetdir, err := filepath.Abs(conf.TargetDir())
	if err != nil {
		return err
	}

	os.MkdirAll(hostdir, os.ModePerm)
	os.MkdirAll(targetdir+"/destdir", os.ModePerm)
	os.MkdirAll(targetdir+"/builddir", os.ModePerm)

	volume_args := []string{}
	for _, dir := range []string{"bin", "etc", "lib", "lib64", "sbin", "tmp", "usr", "var", "builddir", "destdir"} {
		_, err := os.Stat(filepath.Join(targetdir, dir))
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else {
			volume_args = append(volume_args, "-v")
			volume_args = append(volume_args, fmt.Sprintf("%s/%s:/%s", targetdir, dir, dir))
		}
	}

	args := []string{"run", "--rm", "-i", "-v", hostdir + ":/hostdir", "-v", basepath + "/etc:/etc/x10:ro"}
	args = append(args, volume_args...)
	args = append(args, additional_podman_args...)
	args = append(args, "x10_base", "/usr/bin/bash", "-e", "-x")
	logger.Debug(args)
	cmd := exec.Command("podman", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Start()

	stdout_lines := []string{}
	stderr_lines := []string{}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			logger.Debug("[stdout] " + scanner.Text())
			stdout_lines = append(stdout_lines, scanner.Text())
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logger.Debug("[stderr] " + scanner.Text())
			stderr_lines = append(stderr_lines, scanner.Text())
		}
	}()

	stdin.Write([]byte(script + "\n"))
	stdin.Close()
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		logger.Error("Stage failed.")
		logger.Error("Failing stage stdout output is:")
		for _, line := range stdout_lines {
			logger.Error("  " + line)
		}
		logger.Error("Failing stage stderr output is:")
		for _, line := range stderr_lines {
			logger.Error("  " + line)
		}
		return err
	}

	return err
}
