package initial

import (
	"time"

	_ "github.com/yockii/qscore/pkg/database"
)

func init() {
	time.Local = time.FixedZone("CST", 8*3600)
	InitQuestionBankDB()
}
