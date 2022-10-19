package service

import (
	"context"

	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
)

var QuestionBankService = new(questionBankService)

type questionBankService struct{}

func (s *questionBankService) SearchAnswer(question string) string {
	db := GetQuestionBankDb()
	if db == nil {
		return ""
	}
	ctx := context.Background()
	db.BindContextDBConnection(ctx)

	var answer string
	has, err := zorm.QueryRow(
		ctx,
		zorm.NewSelectFinder("tiku", "answer").Append("where question like ?", question+"%"),
		&answer,
	)
	if err != nil {
		logger.Errorln(err)
		return ""
	}
	if !has || answer == "" {
		has, err = zorm.QueryRow(
			ctx,
			zorm.NewSelectFinder("tikuNet", "answer").Append("where question like ?", question+"%"),
			&answer,
		)
		if err != nil {
			logger.Errorln(err)
			return ""
		}
	}
	return answer
}
