package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/domain"

	"xxqg-automate/internal/model"
)

var JobService = new(jobService)

type jobService struct{}

func (_ *jobService) Save(ctx context.Context, job *model.Job) error {
	// activity对象指针不能为空
	if job == nil {
		return errors.New("job对象指针不能为空")
	}

	job.CreateTime = domain.DateTime(time.Now())

	//匿名函数return的error如果不为nil,事务就会回滚
	_, errSaveJob := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始

		_, errSaveJob := zorm.Insert(ctx, job)

		if errSaveJob != nil {
			return nil, errSaveJob
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errSaveJob != nil {
		errSaveJob = fmt.Errorf("jobService.Save错误:%w", errSaveJob)
		logger.Error(errSaveJob)
		return errSaveJob
	}

	return nil
}
func (_ *jobService) DeleteById(ctx context.Context, id string) error {
	//id不能为空
	if len(id) < 1 {
		return errors.New("id不能为空")
	}

	//匿名函数return的error如果不为nil,事务就会回滚
	_, errDeleteActivity := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始
		finder := zorm.NewDeleteFinder(model.JobTableName).Append(" WHERE id=?", id)
		_, errDeleteActivity := zorm.UpdateFinder(ctx, finder)

		if errDeleteActivity != nil {
			return nil, errDeleteActivity
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errDeleteActivity != nil {
		errDeleteActivity = fmt.Errorf("service.Delete错误:%w", errDeleteActivity)
		logger.Error(errDeleteActivity)
		return errDeleteActivity
	}

	return nil
}
func (_ *jobService) FindList(ctx context.Context, finder *zorm.Finder, page *zorm.Page) ([]*model.Job, error) {
	//finder不能为空
	if finder == nil {
		return nil, errors.New("finder不能为空")
	}

	jobList := make([]*model.Job, 0)
	errFindJobList := zorm.Query(ctx, finder, &jobList, page)

	//记录错误
	if errFindJobList != nil {
		errFindJobList = fmt.Errorf("jobService.FindList错误:%w", errFindJobList)
		logger.Error(errFindJobList)
		return nil, errFindJobList
	}

	return jobList, nil
}

func (_ *jobService) DeleteByUserId(ctx context.Context, userId string) error {
	//id不能为空
	if len(userId) < 1 {
		return errors.New("id不能为空")
	}

	//匿名函数return的error如果不为nil,事务就会回滚
	_, errDeleteActivity := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始
		finder := zorm.NewDeleteFinder(model.JobTableName).Append(" WHERE user_id=?", userId)
		_, errDeleteActivity := zorm.UpdateFinder(ctx, finder)

		if errDeleteActivity != nil {
			return nil, errDeleteActivity
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errDeleteActivity != nil {
		errDeleteActivity = fmt.Errorf("service.Delete错误:%w", errDeleteActivity)
		logger.Error(errDeleteActivity)
		return errDeleteActivity
	}

	return nil
}
