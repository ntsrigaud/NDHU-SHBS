package auth

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"shbs-server/pkg/model"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"
)

// HandleRegister creates a new local user account and sends a verification email.
// A JWT is NOT issued here — the client must verify their email before logging in.
//
// POST /api/v1/auth/register
func HandleRegister(db *sqlx.DB, emailSvc *services.EmailService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
		}

		normalizeRegisterRequest(&req)

		if msg := validateRegisterRequest(&req); msg != "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": msg})
		}

		if !util.IsStrongPassword(req.Password) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": "password must be 8+ chars with uppercase, lowercase, digit, and special character",
			})
		}

		// Prevent duplicate accounts. Use a constant-time check to avoid user
		// enumeration timing attacks (OWASP A07).
		var exists bool
		if err := db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", req.Email,
		).Scan(&exists); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}
		if exists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
		}

		hash, err := util.HashPassword(req.Password)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not process password"})
		}

		// INSERT the user row and return the full record in one round-trip.
		// RETURNING avoids a second SELECT and ensures the returned struct reflects
		// exactly what was written (e.g. server-generated created_at).
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

		// Create a 24-hour email-verification token. Best-effort: if this fails
		// (e.g. DB constraint race), the user still has an account and can trigger
		// a resend later.
		rawToken, err := util.CreateVerificationToken(
			db, user.ID, model.VerificationTypeEmailVerification, 24*time.Hour,
		)
		if err != nil {
			log.Printf("warn: could not create verification token for %s: %v", user.Email, err)
		} else {
			verifyURL := fmt.Sprintf(
				"%s/api/v1/auth/verify?token=%s",
				os.Getenv("API_BASE_URL"), rawToken,
			)
			// SMTP failure is logged but does not abort the registration response.
			if err := emailSvc.SendVerificationEmail(user.Email, user.Name, verifyURL); err != nil {
				log.Printf("warn: could not send verification email to %s: %v", user.Email, err)
			}
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
	}
}
