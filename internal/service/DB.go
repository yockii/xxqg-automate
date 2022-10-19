package service

import (
	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/database"

	"xxqg-automate/internal/constant"
)

func GetQuestionBankDb() *zorm.DBDao {
	dao, ok := database.DbMap[constant.QuestionBankSourceName]
	if !ok {
		dao, ok = database.DbMap[constant.QuestionBankSourceName]
		if !ok {
			logger.Errorln("无法成功获取题库数据库连接")
			return nil
		}
	}
	return dao
}
