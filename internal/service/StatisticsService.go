package service

import (
	"context"
	"time"

	"gitee.com/chunanyong/zorm"
	"github.com/sirupsen/logrus"

	"xxqg-automate/internal/domain"
	"xxqg-automate/internal/model"
)

func GetStatisticsInfo() *domain.StatisticsInfo {
	var finished []*model.User
	finder := zorm.NewSelectFinder(model.UserTableName).Append("WHERE last_finish_time>?", time.Now().Format("2006-01-02"))
	err := zorm.Query(context.Background(), finder, &finished, nil)
	if err != nil {
		logrus.Error(err)
	}

	var studying []*model.User
	finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE id in (SELECT user_id FROM " + model.JobTableName + ")")
	err = zorm.Query(context.Background(), finder, &studying, nil)
	if err != nil {
		logrus.Error(err)
	}

	var expired []*model.User
	finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE status=-1")
	err = zorm.Query(context.Background(), finder, &expired, nil)
	if err != nil {
		logrus.Error(err)
	}

	var waiting []*model.User
	finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE status>0 and last_study_time<?", time.Now().Format("2006-01-02"))
	err = zorm.Query(context.Background(), finder, &waiting, nil)
	if err != nil {
		logrus.Error(err)
	}

	var notFinished []*model.User
	finder = zorm.NewSelectFinder(model.UserTableName).Append("WHERE last_score=0")
	err = zorm.Query(context.Background(), finder, &notFinished, nil)
	if err != nil {
		logrus.Error(err)
	}

	info := new(domain.StatisticsInfo)
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
	for _, u := range notFinished {
		info.NotFinished = append(info.NotFinished, u.Nick)
	}
	return info
}
