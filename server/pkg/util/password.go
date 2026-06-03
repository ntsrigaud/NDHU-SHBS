package util

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword returns the bcrypt hash of the plaintext password.
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hashed), err
}

// VerifyPassword compares a plaintext password against a stored bcrypt hash.
// Returns nil on match, bcrypt.ErrMismatchedHashAndPassword on mismatch.
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// IsStrongPassword enforces the minimum password policy:
// at least 8 characters, one uppercase, one lowercase, one digit, one special.
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case 'A' <= c && c <= 'Z':
			hasUpper = true
		case 'a' <= c && c <= 'z':
			hasLower = true
		case '0' <= c && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}
