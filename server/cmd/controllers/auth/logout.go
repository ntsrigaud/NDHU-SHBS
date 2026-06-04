package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/util"
)

// HandleLogout blacklists the caller's current JWT and clears the cookie.
// After this call the token is invalid even if it has not yet expired.
func HandleLogout(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Prefer the httpOnly cookie; fall back to Authorization: Bearer.
		rawToken := c.Cookies("jwt")
		if rawToken == "" {
			if auth := c.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
				rawToken = auth[7:]
			}
		}

		if rawToken != "" {
			if claims, err := util.ParseJWT(rawToken); err == nil {
				_ = util.InvalidateToken(db, c, rawToken, claims.ExpiresAt.Time)
			}
		}

		// Clear the cookie regardless of whether we had a valid token.
		c.Cookie(&fiber.Cookie{
			Name:     "jwt",
			Value:    "",
			Expires:  time.Unix(0, 0),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		return c.JSON(fiber.Map{"message": "logged out"})
	}
}
