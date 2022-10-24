package job

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/domain/wan"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/util"
)

func InitCommunication() {
	// 不用task，而采用延迟的方式
	ants.Submit(fetchServerInfo)
}

func fetchServerInfo() {
	time.Sleep(5 * time.Second)
	defer ants.Submit(fetchServerInfo)
	result := new(wan.StatusAsk)
	_, err := util.GetClient().R().
		SetHeader("token", constant.CommunicateHeaderKey).
		SetResult(result).
		Get(config.GetString("communicate.baseUrl") + "/api/v1/status")
	if err != nil {
		logger.Errorln(err)
		return
	}
	if result.NeedLink {
		logger.Debugln("需要新的登录链接")
		var link string
		link, err = study.GetXxqgRedirectUrl()
		if err != nil {
			logger.Error(err)
		} else {
			resp, _ := util.GetClient().R().
				SetHeader("token", constant.CommunicateHeaderKey).
				SetBody(&wan.LinkInfo{Link: link}).Post(config.GetString("communicate.baseUrl") + "/api/v1/newLink")
			logger.Debugln(resp.ToString())
		}
	}
	if result.NeedStatistics {
		// 查询统计信息，今日完成情况
		var finished []*model.User
		finder := zorm.NewSelectFinder(model.UserTableName).Append("WHERE last_finish_time>?", time.Now().Format("2006-01-02"))
		err = zorm.Query(context.Background(), finder, &finished, nil)
		if err != nil {
			logger.Error(err)
		}

		var studying []*model.User
		finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE id in (SELECT user_id FROM " + model.JobTableName + ")")
		err = zorm.Query(context.Background(), finder, &studying, nil)
		if err != nil {
			logger.Error(err)
		}

		var expired []*model.User
		finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE status=-1")
		err = zorm.Query(context.Background(), finder, &expired, nil)
		if err != nil {
			logger.Error(err)
		}

		var waiting []*model.User
		finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE status>0 and last_study_time<?", time.Now().Format("2006-01-02"))
		err = zorm.Query(context.Background(), finder, &waiting, nil)
		if err != nil {
			logger.Error(err)
		}

		info := new(wan.StatisticsInfo)
		for _, u := range finished {
			info.Finished = append(info.Finished, u.Nick)
		}
		for _, u := range studying {
			info.Studying = append(info.Studying, u.Nick)
		}
		for _, u := range expired {
			info.Expired = append(info.Expired, u.Nick)
		}
		for _, u := range waiting {
			info.Waiting = append(info.Waiting, u.Nick)
		}
		util.GetClient().R().
			SetHeader("token", constant.CommunicateHeaderKey).
			SetBody(info).Post(config.GetString("communicate.baseUrl") + "/api/v1/statisticsNotify")
	}
}
