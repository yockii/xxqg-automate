package lan

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yockii/qscore/pkg/server"

	"xxqg-automate/internal/service"
)

func InitRouter() {

	server.Get("/api/v1/redirectXxqg", UserController.AutoLoginXxqg)

	server.Get("/api/v1/getRedirectXxqg", UserController.GetRedirectUrl)

	server.Get("/api/v1/users", UserController.GetUserList)

	server.Get("/api/v1/statisticsInfo", func(ctx *fiber.Ctx) error {
		info := service.GetStatisticsInfo()
		return ctx.JSON(info)
	})
}
