package lan

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/domain"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/constant"
	internalDomain "xxqg-automate/internal/domain"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/service"
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/util"
)

func InitAutoStudy() {
	// 加载已有的job
	loadJobs()
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
			zorm.NewSelectFinder(model.UserTableName).Append(
				"WHERE last_study_time>? and last_study_time<? and (last_finish_time is null or last_finish_time<last_study_time or last_score=0) and status>0",
				time.Now().Format("2006-01-02"),
				lastTime),
			&notFinished,
			nil,
		); err != nil {
			logger.Errorln(err)
		}
		users = append(users, notFinished...)

		// 开始学习
		for _, user := range users {
			if user.Token != "" {
				if ok, _ := study.CheckUserCookie(study.TokenToCookies(user.Token)); ok {
					ants.Submit(func() {
						study.StartStudy(user)
					})
				} else {
					logger.Warnln("用户登录信息已失效", user.Nick)
					zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
						_, err := zorm.UpdateNotZeroValue(ctx, &model.User{
							Id:            user.Id,
							LastCheckTime: domain.DateTime(time.Now()),
							Status:        -1,
						})
						if err != nil {
							return nil, err
						}
						return zorm.Delete(ctx, &model.Job{UserId: user.Id, Status: 1})
					})
					if config.GetString("communicate.baseUrl") != "" {
						if config.GetBool("xxqg.expireNotify") {
							util.GetClient().R().
								SetHeader("token", constant.CommunicateHeaderKey).
								SetBody(&internalDomain.NotifyInfo{
									Nick: user.Nick,
								}).Post(config.GetString("communicate.baseUrl") + "/api/v1/expiredNotify")
						}
					}
				}
			}
		}
	})
}

func loadJobs() {
	jobs, err := service.JobService.FindList(
		context.Background(),
		zorm.NewSelectFinder(model.JobTableName).Append("WHERE status=1"),
		nil,
	)
	if err != nil {
		logger.Errorln(err)
		return
	}
	for _, job := range jobs {
		uid := job.UserId
		user, err := service.UserService.GetById(context.Background(), uid)
		if err != nil {
			logger.Errorln(err)
			continue
		}
		ants.Submit(func() {
			study.StartStudy(user, job)
		})
		time.Sleep(time.Second)
	}
}
