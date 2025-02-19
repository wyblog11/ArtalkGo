package middleware

import (
	"github.com/ArtalkJS/ArtalkGo/server/common"
	"github.com/gofiber/fiber/v2"
)

func AdminOnlyMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !common.CheckIsAdminReq(c) {
			return common.RespError(c, "需要验证管理员身份", common.Map{"need_login": true})
		}

		return c.Next()
	}
}
