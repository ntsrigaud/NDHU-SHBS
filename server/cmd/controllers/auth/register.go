package auth

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"
)

// RegisterUser creates a new local user account and sends a verification email.
//
// @Summary         Register
// @Description     Creates a new user account and sends an email verification link
// @Tags            Auth
// @Accept          json
// @Param           body body model.SwaggerRegisterRequest true "Registration details"
// @Produce         json
// @Success         201 {object} model.SwaggerMessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         409 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              registerUser
// @Router          /auth/register [post]
func RegisterUser(db *sqlx.DB, emailSvc *services.EmailService, c *fiber.Ctx) error {
	var req model.SwaggerRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is required"})
	}
	if req.Email == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "email is required"})
	}
	if !util.IsStrongPassword(req.Password) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "password must be 8+ chars with uppercase, lowercase, digit, and special character",
		})
	}

	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email).Scan(&exists); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}
	if exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
	}

	hash, err := util.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not process password"})
	}

	var user model.User
	err = db.QueryRowx(
		`INSERT INTO users (id, name, email, password_hash, email_verified, is_admin)
		 VALUES ($1, $2, $3, $4, false, false)
		 RETURNING id, name, email, password_hash, is_admin, email_verified,
		           avatar_image_id, cas_id, created_at, updated_at`,
		uuid.New(), req.Name, req.Email, hash,
	).StructScan(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create account"})
	}

	rawToken, err := util.CreateVerificationToken(db, user.ID, model.VerificationTypeEmailVerification, 24*time.Hour)
	if err != nil {
		log.Printf("warn: could not create verification token for %s: %v", user.Email, err)
	} else {
		verifyURL := fmt.Sprintf("%s/api/v1/auth/verify?token=%s", os.Getenv("API_BASE_URL"), rawToken)
		if err := emailSvc.SendVerificationEmail(user.Email, user.Name, verifyURL); err != nil {
			log.Printf("warn: could not send verification email to %s: %v", user.Email, err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "an email was sent to your account, please verify it before logging in"})
}
