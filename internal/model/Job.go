package model

import (
	"gitee.com/chunanyong/zorm"
	"github.com/yockii/qscore/pkg/domain"
)

const JobTableName = "t_job"

type Job struct {
	zorm.EntityStruct
	Id         string          `json:"id,omitempty" column:"id"`
	UserId     string          `json:"userId,omitempty" column:"user_id"`
	status     int             `json:"status,omitempty" column:"status"`
	Score      int             `json:"score,omitempty" column:"score"`
	CreateTime domain.DateTime `json:"createTime" column:"create_time"`
}

func (entity *Job) GetTableName() string {
	return JobTableName
}

func (entity *Job) GetPKColumnName() string {
	return "id"
}
