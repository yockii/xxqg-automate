package job

import (
	"context"
	"math/rand"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/panjf2000/ants/v2"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"
	"github.com/yockii/qscore/pkg/domain"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/constant"
	internalDomain "xxqg-automate/internal/domain/wan"
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
					ants.Submit(func() {
						startStudy(user)
					})
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

func loadJobs() {
	jobs, err := service.JobService.FindList(
		context.Background(),
		zorm.NewSelectFinder(model.JobTableName),
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
			startStudy(user, job)
		})
		time.Sleep(time.Second)
	}
}

func startStudy(user *model.User, jobs ...*model.Job) {
	var job *model.Job
	if len(jobs) == 0 {
		job = &model.Job{
			UserId: user.Id,
			Score:  0,
		}
		service.JobService.DeleteByUserId(context.Background(), user.Id)
		service.JobService.Save(context.Background(), job)
	} else {
		job = jobs[0]
	}
	if time.Time(user.LastStudyTime).After(time.Now()) {
		time.Sleep(time.Time(user.LastStudyTime).Sub(time.Now()))
	} else {
		randomDuration := time.Duration(rand.Intn(1000))
		_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			finder := zorm.NewUpdateFinder(model.UserTableName).Append(
				"last_study_time=?, last_score=?", domain.DateTime(time.Now().Add(randomDuration*time.Second)), 0).
				Append("WHERE id=?", user.Id)
			return zorm.UpdateFinder(ctx, finder)
			//return zorm.UpdateNotZeroValue(ctx, &model.User{
			//	Id:            user.Id,
			//	LastStudyTime: domain.DateTime(time.Now().Add(randomDuration * time.Second)),
			//	LastScore:     0,
			//})
		})
		if err != nil {
			logger.Errorln(err)
			return
		}

		// 随机休眠再开始学习
		time.Sleep(randomDuration * time.Second)
	}

	logger.Infoln(user.Nick, "开始学习")
	study.Core.Learn(user, constant.Article)
	study.Core.Learn(user, constant.Video)
	study.Core.Answer(user, 1)
	study.Core.Answer(user, 2)
	study.Core.Answer(user, 3)

	time.Sleep(5 * time.Second)

	score, _ := study.GetUserScore(study.TokenToCookies(user.Token))
	_, err := zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
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

	// 删除job
	service.JobService.DeleteById(context.Background(), job.Id)

	util.GetClient().R().
		SetHeader("token", constant.CommunicateHeaderKey).
		SetBody(&internalDomain.FinishInfo{
			Nick:  user.Nick,
			Score: score.TodayScore,
		}).Post(config.GetString("communicate.baseUrl") + "/api/v1/finishNotify")
}
