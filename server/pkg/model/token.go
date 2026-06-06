package model

import (
	"time"

	"github.com/google/uuid"
)

// Verification token type values — mirror the DB CHECK constraint.
const (
	VerificationTypeEmailVerification = "email_verification"
	VerificationTypePasswordReset     = "password_reset"
)

// TokenBlacklist represents a row in the `token_blacklist` table.
// Tokens are stored as SHA-256 hashes (never plaintext) to limit exposure if
// the database is compromised. The scheduler prunes rows whose expires_at has
// passed, keeping the table small.
type TokenBlacklist struct {
	ID        uuid.UUID `db:"id"         json:"-"`
	TokenHash string    `db:"token_hash" json:"-"`
	ExpiresAt time.Time `db:"expires_at" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}

// VerificationToken represents a row in the `verification_tokens` table.
// Used for both email verification and password-reset flows. The raw token is
// emailed to the user; only the SHA-256 hash is stored here (same pattern as
// token_blacklist).
type VerificationToken struct {
	ID        uuid.UUID  `db:"id"         json:"-"`
	UserID    uuid.UUID  `db:"user_id"    json:"-"`
	TokenHash string     `db:"token_hash" json:"-"`
	Type      string     `db:"type"       json:"-"`
	ExpiresAt time.Time  `db:"expires_at" json:"-"`
	UsedAt    *time.Time `db:"used_at"    json:"-"` // NULL until consumed
	CreatedAt time.Time  `db:"created_at" json:"-"`
}
