package auth

import (
	"net/mail"
	"strings"
	"time"

	"shbs-server/pkg/model"
)

// ── Request types ─────────────────────────────────────────────────────────────

// RegisterRequest is the JSON body for POST /api/v1/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ── Response types ────────────────────────────────────────────────────────────

// AuthResponse is returned on a successful login or SSO callback.
// The JWT is also written as an httpOnly cookie by the handler for browser
// clients; the body field serves API and mobile clients.
type AuthResponse struct {
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expires_at"`
	User      model.User `json:"user"`
}

// ── Validation helpers ────────────────────────────────────────────────────────

// normalizeRegisterRequest trims whitespace and lowercases the email in-place.
func normalizeRegisterRequest(r *RegisterRequest) {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

// validateRegisterRequest returns the first validation error found, or nil.
// Keeps handlers clean — all input rules live here in one place.
func validateRegisterRequest(r *RegisterRequest) string {
	if r.Name == "" {
		return "name is required"
	}
	if len(r.Name) > 100 {
		return "name must be 100 characters or fewer"
	}
	if !isValidEmail(r.Email) {
		return "invalid email address"
	}
	if r.Password == "" {
		return "password is required"
	}
	return ""
}

// isValidEmail uses the stdlib mail parser as a lightweight RFC 5322 check.
// net/mail.ParseAddress accepts "Name <email>" format too, so we additionally
// require that the address contains exactly one "@" with content on both sides.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.Split(email, "@")
	return len(parts) == 2 && len(parts[0]) > 0 && strings.Contains(parts[1], ".")
}
