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
	Status     int             `json:"status,omitempty" column:"status"` // 1-学习任务 2-登录任务
	Score      int             `json:"score,omitempty" column:"score"`
	Code       string          `json:"code,omitempty" column:"code"`
	CreateTime domain.DateTime `json:"createTime" column:"create_time"`
}

func (entity *Job) GetTableName() string {
	return JobTableName
}

func (entity *Job) GetPKColumnName() string {
	return "id"
}
