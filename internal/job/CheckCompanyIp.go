package job

import (
	"github.com/yockii/qscore/pkg/task"

	"xxqg-automate/internal/util"
)

func init() {
	task.AddFunc("@every 1h30m", func() {
		util.GetClient().Get("http://192.168.1.8:31558/")
	})
}
