package initial

import (
	"github.com/yockii/qscore/pkg/database"

	"xxqg-automate/internal/constant"
)

func InitQuestionBankDB() {
	database.InitSqlite(constant.QuestionBankSourceName, constant.QuestionBankDBFile)
}
