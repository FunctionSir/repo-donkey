/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2025-07-29 16:22:12
 * @LastEditTime: 2025-08-02 20:54:05
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/build.go
 */

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

func BuildingDir() string {
	return path.Join(Conf.WorkingDir, DIR_BUILDING)
}

func LogsDir() string {
	return path.Join(Conf.WorkingDir, DIR_LOGS)
}

func PkgBuildingDir(pkg *Package) string {
	return path.Join(BuildingDir(), pkg.Name)
}

func PkgLogsDir(pkg *Package) string {
	return path.Join(LogsDir(), pkg.Name)
}

func PkgChrootDir(pkg *Package) string {
	return path.Join(PkgBuildingDir(pkg), DIR_CHROOT)
}

func PkgRootDir(pkg *Package) string {
	return path.Join(PkgChrootDir(pkg), DIR_ROOT)
}

func PkgPkgbuild(pkg *Package) string {
	return path.Join(PkgBuildingDir(pkg), "PKGBUILD")
}

func initWorkingDirs() {
	LogInfo("init working dirs...")
	Check(os.MkdirAll(Conf.WorkingDir, os.ModePerm))
	Check(os.MkdirAll(BuildingDir(), os.ModePerm))
	Check(os.MkdirAll(LogsDir(), os.ModePerm))
	for _, pkg := range Conf.Packages {
		Check(os.MkdirAll(PkgBuildingDir(&pkg), os.ModePerm))
		Check(os.MkdirAll(PkgLogsDir(&pkg), os.ModePerm))
		Check(os.MkdirAll(PkgChrootDir(&pkg), os.ModePerm))
		if !DirExists(PkgRootDir(&pkg)) {
			Check(SudoRun(Conf.BuildUser, Conf.BuildGroup, path.Join(PkgLogsDir(&pkg), LOG_FILE_MKARCHROOT),
				BIN_MKARCHROOT, PkgRootDir(&pkg), "base-devel"))
		}
	}
	LogInfo("all working dirs inited")
}

func GetPkgbuild(pkg *Package) ([]byte, error) {
	if pkg.PKGBUILD == "" {
		LogError("no PKGBUILD specified for " + pkg.Name)
	}
	if strings.HasPrefix(pkg.PKGBUILD, "http://") || strings.HasPrefix(pkg.PKGBUILD, "https://") {
		resp, err := http.Get(pkg.PKGBUILD)
		if err != nil {
			LogWarn("can not get PKGBUILD for package " + pkg.Name + " from the Internet")
			return nil, err
		}
		pkgbuild, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return pkgbuild, nil
	}
	pkgbuild, err := os.ReadFile(pkg.PKGBUILD)
	if err != nil {
		return nil, err
	}
	return pkgbuild, nil
}

func PreBuildPrepare(pkg *Package) (string, bool, error) {
	logFile := path.Join(PkgLogsDir(pkg), strconv.Itoa(int(time.Now().Unix()))+".log")
	if Conf.MakepkgConf != "" {
		makepkgConf := path.Join(PkgRootDir(pkg), CONF_MAKEPKG)
		eq, err := EqualFiles(makepkgConf, Conf.MakepkgConf)
		Check(err)
		if !eq {
			CopyAndOverwrite(makepkgConf, Conf.MakepkgConf)
		}
	}
	if Conf.PacmanConf != "" {
		pacmanConf := path.Join(PkgRootDir(pkg), CONF_PACMAN)
		eq, err := EqualFiles(pacmanConf, Conf.PacmanConf)
		Check(err)
		if !eq {
			CopyAndOverwrite(pacmanConf, Conf.PacmanConf)
		}
	}
	if DirExists(PkgPkgbuild(pkg)) {
		LogError("PKGBUILD of package " + pkg.Name + " exists but is a dir")
	}
	wantedPkgbuild, err := GetPkgbuild(pkg)
	if err != nil {
		LogWarn("can not build package " + pkg.Name + " since can not get PKGBUILD, error is: " + err.Error())
		return logFile, false, err
	}
	if !FileExists(PkgPkgbuild(pkg)) || !PanicOnErr(FileContentIs(PkgPkgbuild(pkg), wantedPkgbuild)) {
		file, err := os.Create(PkgPkgbuild(pkg))
		if err != nil {
			LogWarn("can not build package " + pkg.Name + " since can not write PKGBUILD, error is: " + err.Error())
			return logFile, false, err
		}
		cnt, err := file.Write(wantedPkgbuild)
		if err != nil {
			LogWarn("can not build package " + pkg.Name + " since can not write PKGBUILD, error is: " + err.Error())
			return logFile, false, err
		}
		if Conf.DebugMode {
			LogInfo("written " + strconv.Itoa(cnt) + " bytes to file \"" + PkgPkgbuild(pkg) + "\"")
		}
	} else {
		return logFile, false, nil
	}
	err = SudoRun(Conf.BuildUser, Conf.BuildGroup, logFile, BIN_ARCH_NSPAWN, PkgRootDir(pkg), BIN_PACMAN, "-Syu")
	if err != nil {
		return logFile, false, err
	}
	return logFile, true, nil
}

func BuildPkg(pkg *Package, logFile string) error {
	toRun := make([]string, 0)
	toRun = append(toRun, BIN_BASH)
	toRun = append(toRun, "-c")
	cmdStr := fmt.Sprintf("cd %s;", PkgBuildingDir(pkg)) + " "
	if pkg.BuildProxy != "" {
		cmdStr += fmt.Sprintf("ALL_PROXY=%s HTTP_PROXY=%s HTTPS_PROXY=%s all_proxy=%s http_proxy=%s https_proxy=%s",
			pkg.BuildProxy, pkg.BuildProxy, pkg.BuildProxy, pkg.BuildProxy, pkg.BuildProxy, pkg.BuildProxy) + " "
	}
	cmdStr += BIN_SUDO + " "
	cmdStr += "-u" + Conf.BuildUser + " "
	cmdStr += "-g" + Conf.BuildGroup + " "
	cmdStr += BIN_MAKECHROOTPKG + " "
	cmdStr += "-c -r" + " "
	cmdStr += DIR_CHROOT
	toRun = append(toRun, cmdStr)
	cmd := exec.Command(toRun[0], toRun[1:]...)
	return RunWithLog(cmd, toRun, logFile)
}

func PostBuildOps(pkg *Package, logFile string) error {
	if Conf.PkgSignKey != "" {
		buildingDir, err := os.ReadDir(PkgBuildingDir(pkg))
		if err != nil {
			return err
		}
		for _, e := range buildingDir {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(e.Name(), SUFFIX_PKG) {
				args := make([]string, 0)
				args = append(args, "--sign", "--detach-sign", "--yes")
				if Conf.PkgSignKey != SIGN_USE_DEFAULT {
					args = append(args, "--default-key", Conf.PkgSignKey)
				}
				args = append(args, path.Join(PkgBuildingDir(pkg), e.Name()))
				SudoRun(Conf.BuildUser, Conf.BuildGroup, logFile, BIN_GPG, args...)
			}
		}
	}
	buildingDir, err := os.ReadDir(PkgBuildingDir(pkg))
	if err != nil {
		return err
	}
	for _, e := range buildingDir {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), SUFFIX_PKG) || strings.HasSuffix(e.Name(), SUFFIX_SIG) {
			err := CopyAndOverwrite(path.Join(path.Dir(Conf.TargetDB), e.Name()), path.Join(PkgBuildingDir(pkg), e.Name()))
			if err != nil {
				return err
			}
			err = os.Remove(path.Join(PkgBuildingDir(pkg), e.Name()))
			if err != nil {
				return err
			}
		}
		if !strings.HasSuffix(e.Name(), SUFFIX_PKG) {
			continue
		}
		toRun := make([]string, 0)
		toRun = append(toRun, BIN_REPO_ADD, "--remove")
		switch Conf.PkgSignKey {
		case "":
			LogInfo("will not going to check and sign DB since no key specified")
		case SIGN_USE_DEFAULT:
			toRun = append(toRun, "--verify", "--sign")
		default:
			toRun = append(toRun, "--verify", "--sign", "--key", Conf.PkgSignKey)
		}
		toRun = append(toRun, Conf.TargetDB, path.Join(path.Dir(Conf.TargetDB), e.Name()))
		err := SudoRun(Conf.BuildUser, Conf.BuildGroup, logFile, toRun[0], toRun[1:]...)
		if err != nil {
			return err
		}
	}
	return nil
}
