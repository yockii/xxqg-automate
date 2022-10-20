package job

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/domain"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/model"
	"xxqg-automate/internal/study"
)

func init() {
	task.AddFunc("@every 1m", keepAlive)
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
		study.GetUserScore(study.TokenToCookies(user.Token))
		zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			return zorm.UpdateNotZeroValue(ctx, &model.User{
				Id:            user.Id,
				LastCheckTime: domain.DateTime(time.Now()),
			})
		})
	}
}
