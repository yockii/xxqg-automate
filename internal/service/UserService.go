package service

import (
	"context"
	"errors"
	"fmt"

	"gitee.com/chunanyong/zorm"
	logger "github.com/sirupsen/logrus"
	"github.com/yockii/qscore/pkg/config"

	"xxqg-automate/internal/constant"
	internalDomain "xxqg-automate/internal/domain/wan"
	"xxqg-automate/internal/model"
	"xxqg-automate/internal/util"
)

var UserService = new(userService)

type userService struct{}

func (s *userService) UpdateNotZero(ctx context.Context, user *model.User) error {
	// manager对象指针或主键Id不能为空
	if user == nil || len(user.Id) < 1 {
		return errors.New("user对象指针或主键Id不能为空")
	}

	//匿名函数return的error如果不为nil,事务就会回滚
	_, errUpdateUser := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {

		//事务下的业务代码开始
		_, errUpdateUser := zorm.UpdateNotZeroValue(ctx, user)

		if errUpdateUser != nil {
			return nil, errUpdateUser
		}

		return nil, nil
		//事务下的业务代码结束

	})

	//记录错误
	if errUpdateUser != nil {
		errUpdateUser = fmt.Errorf("更新用户非空值错误:%w", errUpdateUser)
		logger.Error(errUpdateUser)
		return errUpdateUser
	}

	return nil
}

func (_ *userService) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}
	finder := zorm.NewSelectFinder(model.UserTableName).Append(" WHERE username=?", username)
	user := model.User{}
	has, errFindUserByUsername := zorm.QueryRow(ctx, finder, &user)

	// 记录错误
	if errFindUserByUsername != nil {
		errFindUserByUsername = fmt.Errorf("service.FindByUsername错误:%w", errFindUserByUsername)
		logger.Error(errFindUserByUsername)
		return nil, errFindUserByUsername
	}

	if !has {
		return nil, nil
	}
	return &user, nil
}

func (s *userService) GetById(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, errors.New("id不能为空")
	}
	finder := zorm.NewSelectFinder(model.UserTableName).Append("WHERE id=?", id)
	user := new(model.User)
	has, err := zorm.QueryRow(ctx, finder, user)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	if has {
		return user, nil
	}
	return nil, nil
}

func (s *userService) UpdateByUid(ctx context.Context, user *model.User) {
	finder := zorm.NewSelectFinder(model.UserTableName, "count(*)").Append("WHERE uid=?", user.Uid)
	var c int64
	if has, err := zorm.QueryRow(ctx, finder, &c); err != nil {
		logger.Error(err)
		return
	} else if !has {
		return
	}
	_, err := zorm.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		if c == 0 {
			// 新增
			user.Status = 1
			return zorm.Insert(ctx, user)
		} else {
			finder = zorm.NewUpdateFinder(model.UserTableName).
				Append("token=?, login_time=?, status=1", user.Token, user.LoginTime).Append("WHERE uid=?", user.Uid)
			return zorm.UpdateFinder(ctx, finder)
		}
	})
	if err != nil {
		logger.Errorln(err)
		return
	}
	util.GetClient().R().
		SetHeader("token", constant.CommunicateHeaderKey).
		SetBody(&internalDomain.ExpiredInfo{
			Nick: user.Nick,
		}).Post(config.GetString("communicate.baseUrl") + "/api/v1/loginSuccessNotify")
}
