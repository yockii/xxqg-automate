package job

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/domain"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/constant"
	internalDomain "xxqg-automate/internal/domain/wan"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/util"
)

func init() {
	task.AddFunc("0 0/3 6-23 * * *", func() {
		// 1、查出需要执行学习的用户
		lastTime := time.Now().Add(-20 * time.Hour)
		var users []*model.User
		if err := zorm.Query(context.Background(),
			zorm.NewSelectFinder(model.UserTableName).Append("WHERE (last_study_time is null or last_study_time<?) and status>0", lastTime),
			&users,
			nil,
		); err != nil {
			logger.Errorln(err)
			return
		}

		// 查出间隔1小时以上未完成学习的，重新学习
		lastTime = time.Now().Add(-1 * time.Hour)
		var notFinished []*model.User
		if err := zorm.Query(context.Background(),
			zorm.NewSelectFinder(model.UserTableName).Append("WHERE last_study_time>? and last_study_time<? and (last_finish_time is null or last_finish_time<last_study_time)", time.Now().Format("2006-01-02"), lastTime),
			&notFinished,
			nil,
		); err != nil {
			logger.Errorln(err)
		}
		users = append(users, notFinished...)

		// 开始学习
		for _, user := range users {
			if user.Token != "" {
				// 看看2小时内有没有job正在运行
				if ok, _ := study.CheckUserCookie(study.TokenToCookies(user.Token)); ok {
					go startStudy(user)
				} else {
					logger.Warnln("用户登录信息已失效", user.Nick)
					zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
						zorm.UpdateNotZeroValue(ctx, &model.User{
							Id:     user.Id,
							Status: -1,
						})
						return nil, nil
					})
					util.GetClient().R().
						SetHeader("token", constant.CommunicateHeaderKey).
						SetBody(&internalDomain.ExpiredInfo{
							Nick: user.Nick,
						}).Post(config.GetString("communicate.baseUrl") + "/api/v1/expiredNotify")
				}
			}
		}
	})
}

func startStudy(user *model.User) {
	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		return zorm.UpdateNotZeroValue(ctx, &model.User{
			Id:            user.Id,
			LastStudyTime: domain.DateTime(time.Now()),
		})
	})
	if err != nil {
		logger.Errorln(err)
		return
	}
	logger.Infoln(user.Nick, "开始学习")
	study.Core.Learn(user, constant.Article)
	study.Core.Learn(user, constant.Video)
	study.Core.Answer(user, 1)
	study.Core.Answer(user, 2)
	study.Core.Answer(user, 3)

	time.Sleep(5 * time.Second)

	score, _ := study.GetUserScore(study.TokenToCookies(user.Token))
	_, err = zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
		return zorm.UpdateNotZeroValue(ctx, &model.User{
			Id:             user.Id,
			LastCheckTime:  domain.DateTime(time.Now()),
			LastFinishTime: domain.DateTime(time.Now()),
			LastScore:      score.TodayScore,
			Score:          score.TotalScore,
		})
	})
	if err != nil {
		logger.Errorln(err)
		return
	}
	util.GetClient().R().
		SetHeader("token", constant.CommunicateHeaderKey).
		SetBody(&internalDomain.FinishInfo{
			Nick:  user.Nick,
			Score: score.TodayScore,
		}).Post(config.GetString("communicate.baseUrl") + "/api/v1/finishNotify")
}
