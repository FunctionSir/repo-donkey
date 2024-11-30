/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2024-11-16 23:40:46
 * @LastEditTime: 2024-11-30 21:50:38
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/main.go
 */

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/fatih/color"
	"github.com/robfig/cron/v3"
	"gopkg.in/ini.v1"
)

const VER string = "0.0.2"
const CODENAME string = "UiharuKazari"

// For each task.
type Task struct {
	Package       string
	Schedule      string
	CloneDir      string
	TargetDB      string
	Proxy         string
	User          string
	Group         string
	UseChroot     bool
	Sign          bool
	SkipInitBuild bool
}

// Log Dir.
// This should not be changed after config is loaded, or might cause data race!
var LogDir string = ""

// Config loaded.
// This should not be changed after config is loaded, or might cause data race!
var Config *ini.File = nil

// Tasks.
// This should not be changed after config is loaded, or might cause data race!
var Tasks []Task = nil

// Skip init build.
// This should not be changed after config is loaded, or might cause data race!
var SkipInitBuild bool = false

// Debug mode or not.
// This should not be changed after config is loaded, or might cause data race!
var DebugMode bool = false

// For tmp files
var TmpFiles []string

// Locks
var TmpFilesLock sync.Mutex
var ParuLock sync.Mutex
var RepoAddLock sync.Mutex

func LogFatalln(s string) {
	c := color.New(color.FgHiRed, color.Underline)
	log.Fatalln(c.Sprint(s))
}

func LogWarnln(s string) {
	c := color.New(color.FgHiYellow)
	log.Println(c.Sprint(s))
}

func LogInfoln(s string) {
	c := color.New(color.FgHiGreen)
	log.Println(c.Sprint(s))
}

func getStdout(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	output, err := cmd.Output()
	return string(output), err
}

func chkDependences() {
	deps := []string{"paru", "sudo", "repo-add", "gpg", "bash", "proxychains"}
	for _, x := range deps {
		tmp, err := getStdout("whereis", "-b", x)
		if err != nil {
			LogFatalln("CheckDependences: dependence " + x + " check failed: " + err.Error())
		}
		locatedFile := strings.Split(strings.TrimSpace(strings.Split(tmp, ":")[1]), " ")[0]
		if locatedFile == "" {
			LogFatalln("CheckDependences: dependence " + x + " check failed: command " + x + " not found")
		}
		LogInfoln("CheckDependences: dependence " + x + " found at " + locatedFile)
	}
}

func ValueToBool(s string) (bool, error) {
	if s == "true" || s == "True" || s == "T" || s == "t" || s == "1" {
		return true, nil
	}
	if s == "false" || s == "False" || s == "F" || s == "f" || s == "0" {
		return false, nil
	}
	return false, errors.New("ValueToBool: illegal string " + s)
}

func printHelp() {
	fmt.Println("Help of repo-donkey:")
	fmt.Println("-c --config: Specify config file to use.")
	fmt.Println("-l --logdir: Specify the dir that will put logs in.")
	fmt.Println("--no-color: Disable color output (may not affect paru or others).")
	fmt.Println("-s --sikp-init-build: Skip init build.")
	fmt.Println("--debug: Enable debug features.")
	fmt.Println("-h --help: Print this help.")
	fmt.Println("Any unknown args will be ignored with no warning.")
	fmt.Println("To learn more, read the README file (on GitHub or stored in your computer).")
	os.Exit(0)
}

func vitalKeyOrGeneral(sec *ini.Section, general *ini.Section, key string) string {
	if sec.HasKey(key) {
		return sec.Key(key).String()
	}
	if general == nil || !general.HasKey(key) {
		LogFatalln("No key \"" + key + "\" found in section \"" + sec.Name() + "\" or \"GENERAL\" but key \"" + key + "\" is necessary")
	}
	return general.Key(key).String()
}

func getConfig() {
	var confFile string = ""
	Tasks = make([]Task, 0)
	for i, x := range os.Args {
		switch x {
		case "-c", "--config":
			if i+1 >= len(os.Args) {
				LogFatalln("No config file specified")
			}
			confFile = os.Args[i+1]
		case "--no-color":
			color.NoColor = true
		case "-l", "--log-dir":
			if i+1 >= len(os.Args) {
				LogFatalln("No log dir specified")
			}
			LogDir = os.Args[i+1]
		case "-s", "--sikp-init-build":
			SkipInitBuild = true
		case "--debug":
			DebugMode = true
		case "-h", "--help":
			printHelp()
		}
	}
	if confFile == "" {
		LogFatalln("No config file specified")
	}
	var err error
	Config, err = ini.Load(confFile)
	if err != nil {
		LogFatalln("Error occurred during loading the config file: " + err.Error())
	}
	var general *ini.Section = nil
	if Config.HasSection("GENERAL") {
		general = Config.Section("GENERAL")
	}
	for _, sec := range Config.Sections() {
		// Skip section DEFAULT //
		if sec.Name() == "DEFAULT" || sec.Name() == "GENERAL" {
			continue
		}
		var curTask Task
		// Name //
		curTask.Package = sec.Name()
		// Schedule //
		curTask.Schedule = vitalKeyOrGeneral(sec, general, "Schedule")
		if len(strings.Split(curTask.Schedule, " ")) == 5 {
			curTask.Schedule = "0 " + curTask.Schedule
		}
		// Clone dir //
		curTask.CloneDir = vitalKeyOrGeneral(sec, general, "CloneDir")
		// Target DB //
		curTask.TargetDB = vitalKeyOrGeneral(sec, general, "TargetDB")
		// User //
		curTask.User = vitalKeyOrGeneral(sec, general, "User")
		// Group //
		curTask.Group = vitalKeyOrGeneral(sec, general, "Group")
		// Proxy config //
		curTask.Proxy = ""
		if general != nil && general.HasKey("Proxy") {
			curTask.Proxy = general.Key("Proxy").String()
		}
		if sec.HasKey("Proxy") {
			curTask.Proxy = sec.Key("Proxy").String()
			curTask.Proxy = strings.ReplaceAll(curTask.Proxy, "://", " ")
			curTask.Proxy = strings.ReplaceAll(curTask.Proxy, ":", " ")
		}
		// Use chroot or not //
		curTask.UseChroot = false
		if general != nil && general.HasKey("UseChroot") {
			tmpStr := general.Key("UseChroot").String()
			tmp, err := ValueToBool(tmpStr)
			if err != nil {
				LogFatalln("Value of key \"UseChroot\" in \"GENERAL\" can not be parsed: " + err.Error())
			}
			curTask.UseChroot = tmp
		}
		if sec.HasKey("UseChroot") {
			tmp, err := ValueToBool(sec.Key("UseChroot").String())
			if err != nil {
				LogFatalln("Value of key \"UseChroot\" can not be parsed: " + err.Error())
			}
			curTask.UseChroot = tmp
		}
		// Sign or not //
		curTask.Sign = false
		if general != nil && general.HasKey("Sign") {
			tmpStr := general.Key("Sign").String()
			tmp, err := ValueToBool(tmpStr)
			if err != nil {
				LogFatalln("Value of key \"Sign\" in \"GENERAL\" can not be parsed: " + err.Error())
			}
			curTask.Sign = tmp
		}
		if sec.HasKey("Sign") {
			tmp, err := ValueToBool(sec.Key("Sign").String())
			if err != nil {
				LogFatalln("Value of key \"Sign\" can not be parsed: " + err.Error())
			}
			curTask.Sign = tmp
		}
		// Skip initial build //
		curTask.SkipInitBuild = false
		if general != nil && general.HasKey("SkipInitBuild") {
			tmpStr := general.Key("SkipInitBuild").String()
			tmp, err := ValueToBool(tmpStr)
			if err != nil {
				LogFatalln("Value of key \"SkipInitBuild\" in \"GENERAL\" can not be parsed: " + err.Error())
			}
			curTask.SkipInitBuild = tmp
		}
		if sec.HasKey("SkipInitBuild") {
			tmp, err := ValueToBool(sec.Key("SkipInitBuild").String())
			if err != nil {
				LogFatalln("Value of key \"SkipInitBuild\" can not be parsed: " + err.Error())
			}
			curTask.SkipInitBuild = tmp
		}
		// Append to list //
		Tasks = append(Tasks, curTask)
	}
	if LogDir == "" {
		LogFatalln("LogDir not specified, do not know where to put logs")
	}
}

func writeFailLog(task string, combinedOutput []byte) {
	logFile := path.Join(LogDir, task+".fail.latest.log")
	err := os.WriteFile(logFile, combinedOutput, os.ModePerm)
	if err != nil {
		LogWarnln("Can not write log file \"" + logFile + "\": " + err.Error())
	}
}

func builder(task *Task) {
	var cmd *exec.Cmd
	// Build the package //
	program := "sudo"
	args := make([]string, 0)
	LogInfoln("Job for package \"" + task.Package + "\" started")
	if task.Proxy != "" {
		// Gen and write proxychains config.
		proxyConf := "[ProxyList]\n" + task.Proxy + "\n"
		proxyConfPath := "/dev/shm/rd." + task.Package + ".proxy.conf"
		err := os.WriteFile(proxyConfPath, []byte(proxyConf), os.ModePerm)
		if err != nil {
			LogFatalln("Can not generate proxy config file when building package \"" + task.Package + "\": " + err.Error())
		}
		// Add config to tmp file list.
		TmpFilesLock.Lock()
		TmpFiles = append(TmpFiles, proxyConfPath)
		TmpFilesLock.Unlock()
		// Change program and add args.
		program = "proxychains"
		args = append(args, "-q", "-f", proxyConfPath, "sudo")
	}
	args = append(args, "-u", task.User, "-g", task.Group, "paru", "--noconfirm", "--noinstall")
	if task.UseChroot {
		args = append(args, "--chroot")
	} else {
		args = append(args, "--nochroot")
	}
	if task.Sign {
		args = append(args, "--sign")
	} else {
		args = append(args, "--nosign")
	}
	args = append(args, "--clonedir", task.CloneDir, "-S", task.Package)
	cmd = exec.Command(program, args...)
	ParuLock.Lock()
	outputAsBytes, err := cmd.CombinedOutput()
	ParuLock.Unlock()
	if DebugMode {
		LogInfoln(program + " " + strings.Join(args, " ") + " done")
		fmt.Print(string(outputAsBytes))
	}
	if err != nil {
		LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
		writeFailLog(task.Package, outputAsBytes)
		return
	}
	// Copy gened package //
	buildBase := path.Join(task.CloneDir, task.Package)
	entries, err := os.ReadDir(buildBase)
	if err != nil {
		LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
		return
	}
	genedFiles := make([]string, 0)
	for _, x := range entries {
		var flagPkg bool = false
		var flagSig bool = false
		if !x.IsDir() {
			var err error
			conA, err := regexp.MatchString("^"+task.Package+".*.pkg.tar.zst$", x.Name())
			if err != nil {
				LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
				return
			}
			conB, err := regexp.MatchString("^"+task.Package+"-debug.*.pkg.tar.zst$", x.Name())
			if err != nil {
				LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
				return
			}
			flagPkg = conA && !conB
			if task.Sign {
				flagSig, err = regexp.MatchString("^"+task.Package+".*.pkg.tar.zst.sig$", x.Name())
				if err != nil {
					LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
					return
				}
			}
		}
		if flagPkg || flagSig {
			cmd := exec.Command("cp", "-f", "--update=older", path.Join(buildBase, x.Name()), path.Dir(task.TargetDB))
			outputAsBytes, err := cmd.CombinedOutput()
			if DebugMode {
				LogInfoln("cp -f --update=older " + path.Join(buildBase, x.Name()) + " " + path.Dir(task.TargetDB) + " done")
				fmt.Print(string(outputAsBytes))
			}
			if err != nil {
				LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
				writeFailLog(task.Package, outputAsBytes)
				return
			}
		}
		if flagPkg {
			genedFiles = append(genedFiles, path.Join(buildBase, x.Name()))
		}
	}
	// Do repo-add //
	for _, x := range genedFiles {
		args = make([]string, 0)
		args = append(args, "-u", task.User, "-g", task.Group, "repo-add")
		if color.NoColor {
			args = append(args, "--nocolor")
		}
		args = append(args, "-R")
		if task.Sign {
			args = append(args, "-v", "-s")
		}
		args = append(args, task.TargetDB, x)
		cmd = exec.Command("sudo", args...)
		RepoAddLock.Lock()
		outputAsBytes, err := cmd.CombinedOutput()
		RepoAddLock.Unlock()
		if DebugMode {
			LogInfoln("sudo " + strings.Join(args, " ") + " done")
			fmt.Print(string(outputAsBytes))
		}
		if err != nil {
			LogWarnln("Can not finish job for package \"" + task.Package + "\": " + err.Error())
			writeFailLog(task.Package, outputAsBytes)
			return
		}
	}
	LogInfoln("Job for package \"" + task.Package + "\" done")
}

func hello() {
	c := color.New(color.FgHiBlue)
	c.Println("Repo Donkey For Arch Linux [ Version: " + VER + " (" + CODENAME + ") ]")
}

func main() {
	hello()
	getConfig()
	chkDependences()
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)), cron.WithSeconds())
	for _, x := range Tasks {
		c.AddFunc(x.Schedule, func() { builder(&x) })
		LogInfoln("Added task " + x.Package + " targeted to " + path.Base(x.TargetDB) + " with schedule " + x.Schedule)
		if !SkipInitBuild && !x.SkipInitBuild {
			LogInfoln("Started inital build of package " + x.Package)
			builder(&x)
		}
	}
	c.Start()
	s := make(chan (os.Signal), 1)
	signal.Notify(s, syscall.SIGINT)
	<-s
	LogInfoln("Received SIGINT, start to wait for unfinished job to get ready to quit")
	ctx := c.Stop()
	<-ctx.Done()
	LogInfoln("Start to clean tmp files")
	TmpFilesLock.Lock()
	for _, x := range TmpFiles {
		err := os.Remove(x)
		if err != nil {
			LogWarnln("Can not remove tmp file " + x + ": " + err.Error())
		}
	}
	TmpFilesLock.Unlock()
	LogInfoln("All done, exit gracefully")
}
