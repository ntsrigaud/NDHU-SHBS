package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/util"
)

// LoginUser authenticates a local user (email + password) and issues a JWT.
//
// @Summary         Login
// @Description     Authenticates a user with email and password, returns a JWT token
// @Tags            Auth
// @Accept          json
// @Param           body body model.SwaggerLoginRequest true "Login credentials"
// @Produce         json
// @Success         200 {object} model.SwaggerLoginResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              loginUser
// @Router          /auth/login [post]
func LoginUser(db *sqlx.DB, c *fiber.Ctx) error {
	var req model.SwaggerLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user model.User
	if err := db.Get(&user, `SELECT * FROM users WHERE email = $1`, req.Email); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if !user.EmailVerified {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "please verify your email before logging in"})
	}

	if user.PasswordHash == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	if err := util.VerifyPassword(*user.PasswordHash, req.Password); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
	}

	token, expiresAt, err := util.GenerateJWT(user.ID, user.IsAdmin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate token"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return c.JSON(model.SwaggerLoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user,
	})
}
