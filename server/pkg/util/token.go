package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HashToken returns the SHA-256 hex digest of the raw token string.
// Tokens are stored as hashes — never in plaintext — to limit exposure if
// the database is compromised (OWASP A02: Cryptographic Failures).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// InvalidateToken records the hashed token in the blacklist table so it can
// no longer be used after logout, even if it has not yet expired.
func InvalidateToken(db *sqlx.DB, c *fiber.Ctx, tokenString string, expiresAt time.Time) error {
	hash := HashToken(tokenString)
	query := `INSERT INTO token_blacklist (id, token_hash, expires_at) VALUES ($1, $2, $3)
	          ON CONFLICT (token_hash) DO NOTHING`
	if _, err := db.Exec(query, uuid.New(), hash, expiresAt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "could not invalidate token",
		})
	}
	return nil
}

// IsTokenBlacklisted returns true when the token's hash is found in the
// blacklist table with a future expiry (i.e. still within its original TTL).
func IsTokenBlacklisted(db *sqlx.DB, tokenString string) bool {
	hash := HashToken(tokenString)
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM token_blacklist WHERE token_hash = $1 AND expires_at > NOW())`,
		hash,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// DeleteExpiredTokens removes blacklist rows whose tokens have already expired.
// Intended to be called by the scheduler on a periodic interval.
func DeleteExpiredTokens(db *sqlx.DB) {
	if _, err := db.Exec(`DELETE FROM token_blacklist WHERE expires_at <= NOW()`); err != nil {
		log.Printf("Error pruning expired tokens: %v", err)
	}
}

// DeleteExpiredVerificationTokens removes verification tokens past their expiry.
func DeleteExpiredVerificationTokens(db *sqlx.DB) {
	if _, err := db.Exec(`DELETE FROM verification_tokens WHERE expires_at <= NOW()`); err != nil {
		log.Printf("Error pruning expired verification tokens: %v", err)
	}
}

// CreateVerificationToken generates a cryptographically random token, stores
// its SHA-256 hash in the verification_tokens table, and returns the raw token
// string to be embedded in the email link.
//
// The raw token is never stored — only the hash — so a DB breach cannot be
// used to verify accounts or reset passwords without the original email.
func CreateVerificationToken(db *sqlx.DB, userID uuid.UUID, tokenType string, ttl time.Duration) (string, error) {
	// 32 random bytes → 64-char hex string. Enough entropy to be
	// unguessable even if an attacker can enumerate valid user IDs.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token bytes: %w", err)
	}
	rawToken := hex.EncodeToString(raw)
	tokenHash := HashToken(rawToken)
	expiresAt := time.Now().Add(ttl)

	_, err := db.Exec(
		`INSERT INTO verification_tokens (id, user_id, token_hash, type, expires_at)
         VALUES ($1, $2, $3, $4, $5)`,
		uuid.New(), userID, tokenHash, tokenType, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("insert verification token: %w", err)
	}
	return rawToken, nil
}
