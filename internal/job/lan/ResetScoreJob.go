package lan

import (
	"context"

	"gitee.com/chunanyong/zorm"
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/model"
)

func init() {
	task.AddFunc("@daily", func() {
		zorm.Transaction(context.Background(), func(ctx context.Context) (interface{}, error) {
			return zorm.UpdateFinder(ctx, zorm.NewUpdateFinder(model.UserTableName).Append("last_score=0"))
		})
	})
}
