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
	"xxqg-automate/internal/study"
	"xxqg-automate/internal/util"
)

func InitKeepAlive() {
	task.AddFunc("0 0/2 7-22 * * *", keepAlive)
}

func keepAlive() {
	lastTime := time.Now().Add(-4 * time.Hour)
	var users []*model.User
	if err := zorm.Query(context.Background(),
		zorm.NewSelectFinder(model.UserTableName).Append("WHERE (last_check_time is null or last_check_time<?) and status>0", lastTime),
		&users,
		nil,
	); err != nil {
		logger.Error(err)
		return
	}
	for _, user := range users {
		ants.Submit(func() {
			doKeepAlive(user)
		})
	}
}

func doKeepAlive(user *model.User) {
	time.Sleep(time.Duration(rand.Int63n(500)) * time.Second)
	_, failed, err := study.GetUserScore(study.TokenToCookies(user.Token))
	if err != nil {
		logger.Errorln(err)
		return
	}
	if failed {
		zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			return zorm.UpdateNotZeroValue(ctx, &model.User{
				Id:            user.Id,
				LastCheckTime: domain.DateTime(time.Now()),
				Status:        -1,
			})
		})
		util.GetClient().R().
			SetHeader("token", constant.CommunicateHeaderKey).
			SetBody(&internalDomain.ExpiredInfo{
				Nick: user.Nick,
			}).Post(config.GetString("communicate.baseUrl") + "/api/v1/expiredNotify")
	} else {
		zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			return zorm.UpdateNotZeroValue(ctx, &model.User{
				Id:            user.Id,
				LastCheckTime: domain.DateTime(time.Now()),
			})
		})
	}
}
