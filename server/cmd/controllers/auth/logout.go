package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/util"
)

// LogoutUser blacklists the caller's current JWT and clears the cookie.
//
// @Summary         Logout
// @Description     Invalidates the current JWT and clears the session cookie
// @Tags            Auth
// @Produce         json
// @Success         200 {object} model.SwaggerMessageResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @ID              logoutUser
// @Security        BearerAuth
// @Router          /auth/logout [post]
func LogoutUser(db *sqlx.DB, c *fiber.Ctx) error {
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
