package model

import (
	"gitee.com/chunanyong/zorm"
	"github.com/yockii/qscore/pkg/domain"
)

const UserTableName = "t_user"

type User struct {
	zorm.EntityStruct
	Id             string          `json:"id,omitempty" column:"id"`
	Nick           string          `json:"nick,omitempty" column:"nick"`
	Uid            string          `json:"uid,omitempty" column:"uid"`
	Token          string          `json:"token,omitempty" column:"token"`
	LoginTime      int64           `json:"loginTime,omitempty" column:"login_time"`
	LastCheckTime  domain.DateTime `json:"lastCheckTime" column:"last_check_time"`
	LastStudyTime  domain.DateTime `json:"lastStudyTime" column:"last_study_time"`
	LastFinishTime domain.DateTime `json:"lastFinishTime" column:"last_finish_time"`
	LastScore      int             `json:"lastScore" column:"last_score"`
	Score          int             `json:"score" column:"score"`
	Status         int             `json:"status,omitempty" column:"status"`
	CreateTime     domain.DateTime `json:"createTime" column:"create_time"`
}

func (entity *User) GetTableName() string {
	return UserTableName
}

func (entity *User) GetPKColumnName() string {
	return "id"
}
