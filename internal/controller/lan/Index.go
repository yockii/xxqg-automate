package lan

import (
	"github.com/yockii/qscore/pkg/server"
)

func InitRouter() {

	server.Get("/api/v1/redirectXxqg", UserController.AutoLoginXxqg)

	server.Get("/api/v1/getRedirectXxqg", UserController.GetRedirectUrl)
}
