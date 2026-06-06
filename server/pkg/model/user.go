package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a row in the `users` table.
// A user can authenticate via NDHU CAS SSO (cas_id set) or local email/password
// (password_hash set). Both paths coexist — a CAS user who also sets a local
// password will have both fields populated.
type User struct {
	ID            uuid.UUID  `db:"id"              json:"id"`
	Name          string     `db:"name"            json:"name"`
	Email         string     `db:"email"           json:"email"`
	PasswordHash  *string    `db:"password_hash"   json:"-"`                      // NEVER serialise to JSON
	IsAdmin       bool       `db:"is_admin"        json:"is_admin"`
	AvatarImageID *uuid.UUID `db:"avatar_image_id" json:"avatar_image_id,omitempty"`
	EmailVerified bool       `db:"email_verified"  json:"email_verified"`
	CasID         *string    `db:"cas_id"          json:"cas_id,omitempty"`
	CreatedAt     time.Time  `db:"created_at"      json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"      json:"updated_at"`
}

// ErrorResponse is the standard JSON error body returned by all API error responses.
// Defined here so Swagger annotations can reference it across all controller packages.
type ErrorResponse struct {
	Error string `json:"error"`
}
