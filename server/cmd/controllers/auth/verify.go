package auth

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"
)

// VerifyEmail consumes an email-verification token and marks the user account as verified.
//
// @Summary         Verify email
// @Description     Verifies a user's email address using the token from the verification email
// @Tags            Auth
// @Param           token query string true "Verification token"
// @Produce         json
// @Success         200 {object} model.SwaggerMessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              verifyEmail
// @Router          /auth/verify [get]
func VerifyEmail(db *sqlx.DB, c *fiber.Ctx) error {
	rawToken := c.Query("token")
	if rawToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token is required"})
	}

	tokenHash := util.HashToken(rawToken)

	var vt model.VerificationToken
	err := db.QueryRowx(
		`SELECT id, user_id, token_hash, type, expires_at, used_at, created_at
		 FROM verification_tokens WHERE token_hash = $1`,
		tokenHash,
	).StructScan(&vt)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or expired token"})
	}

	if time.Now().After(vt.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token has expired"})
	}
	if vt.UsedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token already used"})
	}
	if vt.Type != model.VerificationTypeEmailVerification {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid token type"})
	}

	tx, err := db.Beginx()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	now := time.Now()
	if _, err := tx.Exec(`UPDATE users SET email_verified = true, updated_at = $1 WHERE id = $2`, now, vt.UserID); err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not verify account"})
	}
	if _, err := tx.Exec(`UPDATE verification_tokens SET used_at = $1 WHERE id = $2`, now, vt.ID); err != nil {
		_ = tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not verify account"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
	}

	frontendURL := os.Getenv("FRONTEND_BASE_URL")
	if frontendURL != "" {
		return c.Redirect(frontendURL+"/login?verified=true", fiber.StatusFound)
	}
	return c.JSON(fiber.Map{"message": "email verified successfully"})
}

// ResendVerification re-sends the verification email for an unverified account.
//
// @Summary         Resend verification email
// @Description     Sends a new verification email to the specified address if the account is unverified
// @Tags            Auth
// @Accept          json
// @Param           body body model.SwaggerResendVerificationRequest true "Email address"
// @Produce         json
// @Success         200 {object} model.SwaggerMessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              resendVerification
// @Router          /auth/resend-verification [post]
func ResendVerification(db *sqlx.DB, emailSvc *services.EmailService, c *fiber.Ctx) error {
	var req model.SwaggerResendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "email is required"})
	}

	var user model.User
	if err := db.Get(&user, `SELECT * FROM users WHERE email = $1`, strings.ToLower(strings.TrimSpace(req.Email))); err != nil {
		// Always return success to prevent email enumeration.
		return c.JSON(fiber.Map{"message": "if that email exists and is unverified, a new link has been sent"})
	}

	if user.EmailVerified {
		return c.JSON(fiber.Map{"message": "if that email exists and is unverified, a new link has been sent"})
	}

	rawToken, err := util.CreateVerificationToken(db, user.ID, model.VerificationTypeEmailVerification, 24*time.Hour)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not create verification token"})
	}

	verifyURL := fmt.Sprintf("%s/api/v1/auth/verify?token=%s", os.Getenv("API_BASE_URL"), rawToken)
	if err := emailSvc.SendVerificationEmail(user.Email, user.Name, verifyURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not send email"})
	}

	return c.JSON(fiber.Map{"message": "if that email exists and is unverified, a new link has been sent"})
}
