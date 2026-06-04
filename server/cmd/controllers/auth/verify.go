package auth

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"shbs-server/pkg/model"
	"shbs-server/pkg/util"
)

// HandleVerify consumes an email-verification token and marks the user account
// as verified. Uses a DB transaction so both updates (user + token) are atomic.
//
// GET /api/v1/auth/verify?token=<raw_token>
func HandleVerify(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rawToken := c.Query("token")
		if rawToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "token is required"})
		}

		// We never stored the raw token — only its SHA-256 hash. Recompute to look up.
		tokenHash := util.HashToken(rawToken)

		var vt model.VerificationToken
		err := db.QueryRowx(
			`SELECT id, user_id, token_hash, type, expires_at, used_at, created_at
			 FROM verification_tokens WHERE token_hash = $1`,
			tokenHash,
		).StructScan(&vt)
		if err != nil {
			// Do not distinguish "not found" from "wrong hash" to prevent oracle attacks.
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

		// Both updates must succeed together. If we mark the user verified but
		// fail to mark the token used, the same link could verify a second account.
		tx, err := db.Beginx()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}

		now := time.Now()
		if _, err := tx.Exec(
			`UPDATE users SET email_verified = true, updated_at = $1 WHERE id = $2`,
			now, vt.UserID,
		); err != nil {
			_ = tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}

		if _, err := tx.Exec(
			`UPDATE verification_tokens SET used_at = $1 WHERE id = $2`,
			now, vt.ID,
		); err != nil {
			_ = tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}

		// Redirect the browser to the frontend login page with a success hint.
		frontendURL := os.Getenv("FRONTEND_BASE_URL")
		return c.Redirect(frontendURL+"/login?verified=true", fiber.StatusFound)
	}
}
