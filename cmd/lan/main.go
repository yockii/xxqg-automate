package main

import (
	"context"
	"flag"
	"os"

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
)

func init() {
	config.DefaultInstance.SetDefault("server.port", 8080)
	config.DefaultInstance.SetDefault("database.type", "sqlite")
	config.DefaultInstance.SetDefault("database.address", "./conf/data.db")
	config.DefaultInstance.SetDefault("logger.level", "debug")
	config.DefaultInstance.SetDefault("xxqg.schema", "https://scintillating-axolotl-8c8432.netlify.app/?")
	config.DefaultInstance.SetDefault("xxqg.expireNotify", false)

	flag.StringVar(&baseUrl, "baseUrl", "", "服务端url")
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
	defer ants.Release()
	logger.Infoln("当前应用版本: " + VERSION)
	logger.Infoln("开始检测环境.....")

	study.Init()
	defer study.Quit()

	database.InitSysDb()

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
		return nil, nil

	})
}

func startWeb() {
	lan.InitRouter()
	logger.Error(server.Start(":" + config.GetString("server.port")))
}
