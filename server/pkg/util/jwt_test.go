package util_test

import (
	"os"
	"testing"
	"time"

	"shbs-server/pkg/util"

	"github.com/google/uuid"
)

func setJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long!")
	t.Setenv("JWT_EXPIRY_HOURS", "1")
}

// ── GenerateJWT / ParseJWT ────────────────────────────────────────────────────

func TestGenerateJWT_ValidToken(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, expiresAt, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateJWT returned empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("ExpiresAt is in the past")
	}
}

func TestParseJWT_ValidClaims(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, true)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}

	claims, err := util.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
	}
	if !claims.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestParseJWT_InvalidToken(t *testing.T) {
	setJWTEnv(t)
	_, err := util.ParseJWT("not.a.valid.token")
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestParseJWT_WrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long!")
	t.Setenv("JWT_EXPIRY_HOURS", "1")

	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}

	// Change secret; the token should no longer be valid.
	os.Setenv("JWT_SECRET", "completely-different-secret-value-xx!!")
	_, err = util.ParseJWT(token)
	if err == nil {
		t.Fatal("Expected error for wrong secret, got nil")
	}
}

func TestGenerateJWT_DefaultExpiry(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret-that-is-at-least-32-chars-long!")
	// No JWT_EXPIRY_HOURS set — should default to 24h.
	os.Unsetenv("JWT_EXPIRY_HOURS")

	userID := uuid.New()
	_, expiresAt, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}
	// Expiry should be roughly 24 hours from now (allow a few seconds of slack).
	expected := time.Now().Add(23 * time.Hour)
	if expiresAt.Before(expected) {
		t.Errorf("ExpiresAt %v is earlier than expected minimum %v", expiresAt, expected)
	}
}
