/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2025-07-28 23:18:36
 * @LastEditTime: 2025-08-01 11:03:29
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/os.go
 */
package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	BIN_SUDO          string = "/usr/bin/sudo"
	BIN_BASH          string = "/usr/bin/bash"
	BIN_MKARCHROOT    string = "/usr/bin/mkarchroot"
	BIN_ARCH_NSPAWN   string = "/usr/bin/arch-nspawn"
	BIN_PACMAN        string = "/usr/bin/pacman"
	BIN_MAKECHROOTPKG string = "/usr/bin/makechrootpkg"
	BIN_GPG           string = "/usr/bin/gpg"
	BIN_REPO_ADD      string = "/usr/bin/repo-add"
	CONF_MAKEPKG      string = "etc/makepkg.conf"
	CONF_PACMAN       string = "etc/pacman.conf"
	SUFFIX_PKG        string = ".pkg.tar.zst"
	SUFFIX_SIG        string = ".pkg.tar.zst.sig"
)

func FileExists(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) || stat.IsDir() {
		return false
	}
	return true
}

func DirExists(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) || !stat.IsDir() {
		return false
	}
	return true
}

func CopyAndOverwrite(dst, src string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	cnt, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	if Conf.DebugMode {
		LogInfo("copied \"" + src + "\" to \"" + dst + "\", " + strconv.Itoa(int(cnt)) + " byte written recorded")
	}
	return nil
}

func PanicOnErr[T any](some T, err error) T {
	if err != nil {
		LogError(err.Error())
	}
	return some
}

func EqualFiles(a, b string) (bool, error) {
	aFile, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	bFile, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(aFile, bFile), nil
}

func FileContentIs(name string, content []byte) (bool, error) {
	file, err := os.ReadFile(name)
	if err != nil {
		return false, err
	}
	return bytes.Equal(file, content), nil
}

func SudoRun(asUser string, asGroup string, logTo string, name string, args ...string) error {
	toRun := make([]string, 0)
	toRun = append(toRun, BIN_SUDO, "-u", asUser, "-g", asGroup, name)
	toRun = append(toRun, args...)
	cmd := exec.Command(toRun[0], toRun[1:]...)
	return RunWithLog(cmd, toRun, logTo)
}

func RunWithLog(cmd *exec.Cmd, toRun []string, logTo string) error {
	if Conf.DebugMode {
		LogInfo("will run command \"" + strings.Join(toRun, " ") + "\"")
	}
	if logTo != "" {
		logFile, err := os.OpenFile(logTo, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			return err
		}
		defer logFile.Close()
		var logWritter io.Writer
		if Conf.DebugMode {
			cmd.Stdin = os.Stdin
			logWritter = io.MultiWriter(bufio.NewWriter(logFile), os.Stdout)
		} else {
			logWritter = bufio.NewWriter(logFile)
		}
		cmd.Stderr = logWritter
		cmd.Stdout = logWritter
	}
	err := cmd.Run()
	if Conf.DebugMode {
		if err == nil {
			LogInfo("command \"" + strings.Join(toRun, " ") + "\" done without error")
		} else {
			LogWarn("command \"" + strings.Join(toRun, " ") + "\" done with error: " + err.Error())
		}
	}
	return err
}
