package lan

import (
	"github.com/gofiber/fiber/v2"

	"xxqg-automate/internal/study"
)

var UserController = new(userController)

type userController struct{}

func (c *userController) AutoLoginXxqg(ctx *fiber.Ctx) error {
	u, err := study.GetXxqgRedirectUrl()
	if err != nil {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	return ctx.Redirect(u)
}

func (c *userController) GetRedirectUrl(ctx *fiber.Ctx) error {
	u, err := study.GetXxqgRedirectUrl()
	if err != nil {
		return ctx.SendStatus(fiber.StatusBadRequest)
	}
	return ctx.SendString(u)
}
