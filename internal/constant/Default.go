package constant

import "github.com/yockii/qscore/pkg/config"

const (
	DefaultRoleId     = "999999999"
	DefaultRoleName   = "超级管理员"
	ResourceTypeRoute = "route"
)

const (
	QuestionBankSourceName = "questionBank"
	QuestionBankDBFile     = "./conf/QuestionBank.db"
)

var (
	CommunicateHeaderKey = config.GetString("server.token")
)
