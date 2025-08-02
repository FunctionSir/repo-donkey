/*
 * @Author: FunctionSir
 * @License: AGPLv3
 * @Date: 2025-07-28 10:56:55
 * @LastEditTime: 2025-08-02 20:51:31
 * @LastEditors: FunctionSir
 * @Description: -
 * @FilePath: /repo-donkey/main.go
 */
package main

import (
	"os"
	"os/signal"
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const FLG_FILE_NO_ERR_BEFORE string = "LAST-BUILD-OK"

var JobsWg sync.WaitGroup

func buildAll(limiter chan struct{}, stop chan struct{}) {
buildloop:
	for index := range Conf.Packages {
		select {
		case <-stop:
			LogInfo("building: graceful exit signal received, no new jobs will be created")
			break buildloop
		default:
			JobsWg.Add(1)
			limiter <- struct{}{}
			go func() {
				defer JobsWg.Done()
				defer func() { <-limiter }()
				LogInfo("will build package " + Conf.Packages[index].Name + "...")
				logFile, changed, err := PreBuildPrepare(&Conf.Packages[index])
				if err != nil {
					LogWarn("can not start to build " + Conf.Packages[index].Name + ": " + err.Error())
					return
				}
				if !changed && FileExists(path.Join(PkgBuildingDir(&Conf.Packages[index]), FLG_FILE_NO_ERR_BEFORE)) {
					LogInfo("skiped the build process of package " + Conf.Packages[index].Name + ": PKGBUILD not changed and no error before")
					return
				}
				err = BuildPkg(&Conf.Packages[index], logFile)
				if err != nil {
					LogWarn("can not build package " + Conf.Packages[index].Name + " properly: " + err.Error())
					return
				}
				err = PostBuildOps(&Conf.Packages[index], logFile)
				if err != nil {
					LogWarn("can not finish post-build process of package " + Conf.Packages[index].Name + ": " + err.Error())
					return
				}
				okFile, err := os.Create(path.Join(PkgBuildingDir(&Conf.Packages[index]), FLG_FILE_NO_ERR_BEFORE))
				if err != nil {
					LogWarn("can not create build-ok flag file: " + err.Error())
					return
				}
				defer okFile.Close()
				cnt, err := okFile.WriteString("DELETE THIS FILE IF YOU WANT TO REBUILD")
				if err != nil {
					LogWarn("can not write build-ok flag file" + err.Error())
					return
				}
				if Conf.DebugMode {
					LogInfo("written " + strconv.Itoa(cnt) + " bytes to file \"" + path.Join(PkgBuildingDir(&Conf.Packages[index]), FLG_FILE_NO_ERR_BEFORE) + "\"")
				}
				LogInfo("the build process of " + Conf.Packages[index].Name + " finished successfully")
			}()
		}
	}
}

func ticker(limiter chan struct{}, stop chan struct{}) {
	tick := time.NewTicker(Conf.Schedule)
	defer tick.Stop()
tickerloop:
	for {
		select {
		case <-stop:
			LogInfo("ticker: graceful exit signal received, will stop the ticker")
			tick.Stop()
			break tickerloop
		case <-tick.C:
			buildAll(limiter, stop)
		}
	}
}

func main() {
	stopSig := make(chan os.Signal, 1)
	stop := make(chan struct{})
	signal.Notify(stopSig, syscall.SIGINT)
	go func() {
		<-stopSig
		LogInfo("signal SIGINT received, will start the graceful stop process...")
		close(stop)
	}()
	getConf()
	initWorkingDirs()
	limiter := make(chan struct{}, Conf.WorkersCnt)
	buildAll(limiter, stop)
	ticker(limiter, stop)
	LogInfo("graceful exit: waiting existing jobs...")
	JobsWg.Wait()
	LogInfo("graceful exit: goodbye!")
}
