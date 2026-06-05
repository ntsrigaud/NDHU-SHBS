package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/util"
)

// HandleLogin authenticates a local user (email + password) and issues a JWT.
//
// Security: we return the same HTTP 401 for "user not found" and "wrong
// password" to prevent user-enumeration attacks (OWASP A07).
func HandleLogin(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		req.Email = strings.ToLower(strings.TrimSpace(req.Email))

		var user model.User
		if err := db.Get(&user, `SELECT * FROM users WHERE email = $1`, req.Email); err != nil {
			// Identical error for "not found" and "wrong password".
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}

		if !user.EmailVerified {
			return fiber.NewError(fiber.StatusForbidden, "please verify your email before logging in")
		}

		// SSO-only accounts have no local password.
		if user.PasswordHash == nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}

		if err := util.VerifyPassword(*user.PasswordHash, req.Password); err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}

		token, expiresAt, err := util.GenerateJWT(user.ID, user.IsAdmin)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not generate token")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		return c.JSON(AuthResponse{
			Token:     token,
			ExpiresAt: expiresAt,
			User:      user,
		})
	}
}
