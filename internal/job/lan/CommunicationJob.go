package lan

import (
	"context"
	"time"
	"xxqg-automate/internal/model"

	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"

	"xxqg-automate/internal/constant"
	"xxqg-automate/internal/domain"
	"xxqg-automate/internal/service"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/util"
)

func InitCommunication() {
	if config.GetString("communicate.baseUrl") != "" {
		// 不用task，而采用延迟的方式
		ants.Submit(fetchServerInfo)
	}
}

func fetchServerInfo() {
	time.Sleep(5 * time.Second)
	defer ants.Submit(fetchServerInfo)
	result := new(domain.StatusAsk)
	_, err := util.GetClient().R().
		SetHeader("token", constant.CommunicateHeaderKey).
		SetResult(result).
		Get(config.GetString("communicate.baseUrl") + "/api/v1/status")
	if err != nil {
		logger.Errorln(err)
		return
	}
	//logger.Debugln(resp.ToString())
	if result.NeedLink {
		logger.Debugln("需要新的登录链接")
		var link string
		link, err = study.GetXxqgRedirectUrl(result.LinkDingId)
		if err != nil {
			logger.Error(err)
		} else {
			resp, _ := util.GetClient().R().
				SetHeader("token", constant.CommunicateHeaderKey).
				SetBody(&domain.LinkInfo{Link: link}).Post(config.GetString("communicate.baseUrl") + "/api/v1/newLink")
			logger.Debugln(resp.ToString())
		}
	}
	if result.NeedStatistics {
		logger.Debugln("需要统计信息")
		// 查询统计信息，今日完成情况
		info := service.GetStatisticsInfo()
		util.GetClient().R().
			SetHeader("token", constant.CommunicateHeaderKey).
			SetBody(info).Post(config.GetString("communicate.baseUrl") + "/api/v1/statisticsNotify")
	}

	if len(result.BindUsers) > 0 {
		// 绑定用户
		for dingtalkId, nick := range result.BindUsers {
			service.UserService.BindUser(nick, dingtalkId)
		}
	}

	if len(result.StartStudy) > 0 {
		// 有需要立即开始学习的
		// DingTalkId的列表
		for _, dingId := range result.StartStudy {
			dingtalkId := dingId
			user, err := service.UserService.FindByDingtalkId(dingtalkId)
			if err != nil {
				logger.Errorln(err)
				continue
			}
			if user != nil {
				_ = ants.Submit(func() {
					study.StartStudyRightNow(user)
				})
			}
		}
	}

	if len(result.TagUsers) > 0 {
		// 进行用户标记
		for _, dingId := range result.StartStudy {
			dingtalkId := dingId
			user, err := service.UserService.FindByDingtalkId(dingtalkId)
			if err != nil {
				logger.Errorln(err)
				continue
			}
			if user != nil {
				_ = ants.Submit(func() {
					_ = service.UserService.UpdateNotZero(context.Background(), &model.User{Id: user.Id, OnlyLoginTag: 1})
				})
			}
		}
	}
}
