package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"gitee.com/chunanyong/zorm"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/database"
	"github.com/yockii/qscore/pkg/server"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/controller/lan"
	job "xxqg-automate/internal/job/lan"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/update"

	_ "xxqg-automate/internal/initial"
)

var VERSION = ""

var (
	baseUrl string
	daemon  bool
)

func init() {
	config.DefaultInstance.SetDefault("server.port", 8080)
	config.DefaultInstance.SetDefault("database.type", "sqlite")
	config.DefaultInstance.SetDefault("database.address", "./conf/data.db")
	config.DefaultInstance.SetDefault("logger.level", "debug")
	config.DefaultInstance.SetDefault("xxqg.schema", "https://scintillating-axolotl-8c8432.netlify.app/?")
	config.DefaultInstance.SetDefault("xxqg.expireNotify", false)

	flag.StringVar(&baseUrl, "baseUrl", "", "服务端url")
	flag.BoolVar(&daemon, "daemon", false, "以守护进程方式启动")
	flag.Parse()

	if baseUrl != "" {
		config.DefaultInstance.Set("communicate.baseUrl", baseUrl)
	}

	// 写入配置文件
	if err := config.DefaultInstance.WriteConfig(); err != nil {
		logger.Errorln(err)
	}

}

func main() {
	if daemon {
		runDaemon(os.Args)
		return
	}

	defer ants.Release()
	logger.Infoln("当前应用版本: " + VERSION)
	logger.Infoln("开始检测环境.....")

	study.Init()
	defer study.Quit()
	database.InitSysDb()
	study.LoadLoginJobs()

	// 检查数据库文件是否存在
	if config.GetString("database.type") == "sqlite" {
		_, err := os.Stat(config.GetString("database.address"))
		if err != nil && os.IsNotExist(err) {
			// 不存在
			f, _ := os.Create(config.GetString("database.address"))
			f.Close()
		}

		// 创建数据库
		createTables()
	}

	ants.Submit(func() {
		update.CheckUpdate(VERSION)
	})

	//if !util.CheckQuestionDB() {
	//	util.DownloadDbFile()
	//}

	job.InitAutoStudy()
	job.InitKeepAlive()
	job.InitCommunication()

	task.Start()
	defer task.Stop()

	startWeb()
}

// 以守护进程方式启动
func runDaemon(args []string) {
	fmt.Printf("pid:%d ppid: %d, arg: %s \n", os.Getpid(), os.Getppid(), os.Args)
	// 去除--daemon参数，启动主程序
	for i := 0; i < len(args); {
		if args[i] == "--daemon" && i != len(args)-1 {
			args = append(args[:i], args[i+1:]...)
		} else if args[i] == "--daemon" && i == len(args)-1 {
			args = args[:i]
		} else {
			i++
		}
	}
	// 启动子进程
	for {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "启动失败, Error: %s \n", err)
			return
		}
		fmt.Printf("守护进程模式启动学习端, pid:%d ppid: %d, arg: %s \n", cmd.Process.Pid, os.Getpid(), args)
		cmd.Wait()
	}
}

func createTables() {
	ctx := context.Background()
	zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		userTable := `create table t_user
(
    id               varchar(50)
        constraint t_user_pk
            primary key,
    nick             varchar(50),
    uid              varchar(50),
    token            varchar(500),
    login_time       INTEGER,
    status           INTEGER,
    create_time      datetime,
    last_check_time  datetime,
    last_study_time  datetime,
    last_finish_time datetime,
    last_score       INTEGER,
    score            INTEGER,
    only_login_tag 	 INTEGER,
    dingtalk_id      varchar(50)
);`
		zorm.UpdateFinder(ctx, zorm.NewFinder().Append(userTable))
		jobTable := `create table t_job
(
    id          varchar(50)
        constraint t_job_pk
            primary key,
    user_id     varchar(50),
    score       INTEGER,
    status      INTEGER,
    code        varchar(50),
    create_time datetime
);`
		zorm.UpdateFinder(ctx, zorm.NewFinder().Append(jobTable))

		alterUserTable := `alter table t_user add column only_login_tag INTEGER;`
		zorm.UpdateFinder(ctx, zorm.NewFinder().Append(alterUserTable))

		return nil, nil
	})
}

func startWeb() {
	lan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}
