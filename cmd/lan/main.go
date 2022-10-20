package main

import (
	"flag"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/huoxue1/xdaemon"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/server"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/controller/lan"
	_ "xxqg-automate/internal/job"
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

	background()

	flag.BoolVar(&u, "update", false, "更新应用")
	flag.Parse()

}

func background() {
	cmd, err := xdaemon.Background(os.Stdout, false)
	if err != nil {
		logger.Fatalln(err)
	}
	if xdaemon.IsParent() {
		go onKill(cmd)
		for {
			_ = cmd.Wait()
			if cmd.ProcessState.Exited() {
				logger.Infoln(cmd.ProcessState)
				if cmd.ProcessState.ExitCode() != 201 {
					break
				} else {
					logger.Infoln("重启应用...")
				}
			}
			cmd, err = xdaemon.Background(os.Stdout, false)
			if err != nil {
				return
			}
		}
		os.Exit(0)
	}
}

func onKill(cmd *exec.Cmd) {
	c := make(chan os.Signal)
	// 监听ctrl+c的信号
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	os.Exit(1)
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

	task.Start()
	defer task.Stop()

	startWeb()
}

func startWeb() {
	lan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}
