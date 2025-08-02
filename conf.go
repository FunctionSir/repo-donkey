/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2025-07-28 21:00:43
 * @LastEditTime: 2025-08-02 20:56:09
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/conf.go
 */

package main

import (
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/FunctionSir/readini"
)

const (
	SEC_GENERAL string = "GENERAL"
)

const (
	KEY_DIR          string = "Dir"
	KEY_TARGET_DB    string = "TargetDB"
	KEY_USER         string = "User"
	KEY_GROUP        string = "Group"
	KEY_WORKERS      string = "Workers"
	KEY_SCHEDULE     string = "Schedule"
	KEY_PROXY        string = "Proxy"
	KEY_KEY          string = "Key"
	KEY_MAKEPKG_CONF string = "MakepkgConf"
	KEY_PACMAN_CONF  string = "PacmanConf"
	KEY_DEBUG_MODE   string = "DebugMode"
	KEY_PKGBUILD     string = "PKGBUILD"
)

const (
	DIR_PKGBUILDS string = "PKGBUILDs"
	DIR_BUILDING  string = "building"
	DIR_LOGS      string = "logs"
	DIR_CHROOT    string = "chroot"
	DIR_ROOT      string = "root"
)

const (
	LOG_FILE_MKARCHROOT string = "mkarchroot.log"
)

const (
	SIGN_USE_DEFAULT string = "DEFAULT"
)

const AUR_URL_BASE string = "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h="

type Package struct {
	Name       string
	PKGBUILD   string
	BuildProxy string
}

type Config struct {
	WorkingDir  string
	TargetDB    string
	BuildUser   string
	BuildGroup  string
	BuildProxy  string
	MakepkgConf string
	PacmanConf  string
	PkgSignKey  string
	WorkersCnt  int
	DebugMode   bool
	Schedule    time.Duration
	Packages    []Package
}

var Conf Config

func chkSection(conf readini.Conf, sec string) {
	if !conf.HasSection(sec) {
		LogError("section \"" + sec + "\" not found in config file")
	}
}

func chkKey(sec readini.Sec, name string, key string) {
	if !sec.HasKey(key) {
		LogError("no key \"" + key + "\" found in section \"" + name + "\"")
	}
}

func ConfValToDuration(val string) time.Duration {
	res, err := time.ParseDuration(val)
	Check(err)
	return res
}

func ConfValToInt(val string) int {
	res, err := strconv.Atoi(val)
	Check(err)
	return res
}

func ConfValToBool(val string) bool {
	if val == "true" || val == "True" || val == "T" || val == "t" || val == "1" {
		return true
	}
	if val == "false" || val == "False" || val == "F" || val == "f" || val == "0" {
		return false
	}
	LogError("can not convert \"" + val + "\" to bool")
	return false
}

func getConf() {
	if len(os.Args) < 2 {
		LogError("no config file specified")
	}

	conf, err := readini.LoadFromFile(os.Args[1])
	Check(err)
	LogInfo("using config file \"" + os.Args[1] + "\"")

	chkSection(conf, SEC_GENERAL)
	sec := conf[SEC_GENERAL]

	chkKey(sec, SEC_GENERAL, KEY_DIR)
	chkKey(sec, SEC_GENERAL, KEY_TARGET_DB)
	chkKey(sec, SEC_GENERAL, KEY_USER)
	chkKey(sec, SEC_GENERAL, KEY_GROUP)

	if !DirExists(path.Dir(sec[KEY_TARGET_DB])) || !strings.HasSuffix(sec[KEY_TARGET_DB], ".db.tar.gz") {
		LogError("invalid path for target database")
	}

	Conf.WorkingDir = sec[KEY_DIR]
	Conf.TargetDB = sec[KEY_TARGET_DB]
	Conf.BuildUser = sec[KEY_USER]
	Conf.BuildGroup = sec[KEY_GROUP]
	Conf.Packages = make([]Package, 0)
	Conf.WorkersCnt = runtime.NumCPU()
	Conf.Schedule = 24 * time.Hour
	Conf.PkgSignKey = ""
	Conf.BuildProxy = ""
	Conf.MakepkgConf = ""
	Conf.PacmanConf = ""
	Conf.DebugMode = false

	if sec.HasKey(KEY_KEY) {
		Conf.PkgSignKey = sec[KEY_KEY]
	}
	if sec.HasKey(KEY_SCHEDULE) {
		Conf.Schedule = ConfValToDuration(sec[KEY_SCHEDULE])
	}
	if sec.HasKey(KEY_PROXY) {
		Conf.BuildProxy = sec[KEY_PROXY]
	}
	if sec.HasKey(KEY_WORKERS) {
		Conf.WorkersCnt = ConfValToInt(sec[KEY_WORKERS])
	}
	if sec.HasKey(KEY_MAKEPKG_CONF) {
		Conf.MakepkgConf = sec[KEY_MAKEPKG_CONF]
	}
	if sec.HasKey(KEY_PACMAN_CONF) {
		Conf.PacmanConf = sec[KEY_PACMAN_CONF]
	}
	if sec.HasKey(KEY_DEBUG_MODE) {
		Conf.DebugMode = ConfValToBool(sec[KEY_DEBUG_MODE])
	}

	for pkgName, pkgConf := range conf {
		curPkg := Package{Name: pkgName, PKGBUILD: AUR_URL_BASE + pkgName, BuildProxy: Conf.BuildProxy}
		if pkgName == SEC_GENERAL || pkgName == "" {
			continue
		}
		if !pkgConf.HasKey(KEY_PKGBUILD) {
			LogInfo("building process of package " + pkgName + " will based on PKGBUILD downloaded from AUR")
		} else {
			curPkg.PKGBUILD = pkgConf[KEY_PKGBUILD]
		}
		if pkgConf.HasKey(KEY_PROXY) {
			curPkg.BuildProxy = pkgConf[KEY_PROXY]
		}
		Conf.Packages = append(Conf.Packages, curPkg)
	}
}
