package middleware

import (
	"shbs-server/pkg/util"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
)

// Auth validates the JWT from the request (cookie or Authorization header),
// checks the token blacklist, and injects the user ID into the Fiber context
// locals under the key "userID".
func Auth(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := util.ExtractClaims(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized: " + err.Error(),
			})
		}

		// Reject blacklisted tokens (issued before logout).
		rawToken := c.Cookies("jwt")
		if rawToken == "" {
			auth := c.Get("Authorization")
			if len(auth) > 7 {
				rawToken = auth[7:]
			}
		}
		if util.IsTokenBlacklisted(db, rawToken) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "token has been revoked",
			})
		}

		c.Locals("userID", claims.UserID.String())
		c.Locals("isAdmin", claims.IsAdmin)
		return c.Next()
	}
}
