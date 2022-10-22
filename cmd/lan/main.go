package main

import (
	"flag"
	"os"

	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/server"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/controller/lan"
	"xxqg-automate/internal/job"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/update"
	"xxqg-automate/internal/util"

	_ "xxqg-automate/internal/initial"
)

var VERSION = ""

var (
	u bool
)

func init() {
	config.DefaultInstance.SetDefault("server.port", 8080)

	flag.BoolVar(&u, "update", false, "更新应用")
	flag.Parse()

}

func main() {
	defer ants.Release()
	logger.Infoln("当前应用版本: " + VERSION)
	logger.Infoln("开始检测环境.....")

	study.Init()
	defer study.Quit()

	ants.Submit(func() {
		update.CheckUpdate(VERSION)
	})
	if u {
		update.SelfUpdate("", VERSION)
		logger.Infoln("请重启应用")
		os.Exit(1)
	}

	if !util.CheckQuestionDB() {
		util.DownloadDbFile()
	}

	job.InitAutoStudy()
	job.InitKeepAlive()
	job.InitCommunication()

	task.Start()
	defer task.Stop()

	startWeb()
}

func startWeb() {
	lan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}
