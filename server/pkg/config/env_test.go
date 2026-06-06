package config_test

import (
	"os"
	"testing"

	"shbs-server/pkg/config"
)

// requiredVars must match what Load checks.
var requiredVars = []string{
	"PORT", "DATABASE_URL", "POSTGRES_USER", "POSTGRES_PASSWORD",
	"POSTGRES_DB", "JWT_SECRET", "JWT_EXPIRY_HOURS",
	"API_BASE_URL", "FRONTEND_BASE_URL", "AI_SERVICE_URL",
}

func setAllRequired(t *testing.T) {
	t.Helper()
	t.Setenv("PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://x:x@localhost:5432/x?sslmode=disable")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_DB", "shbs")
	t.Setenv("JWT_SECRET", "a-sufficiently-long-secret-for-testing-purposes!!")
	t.Setenv("JWT_EXPIRY_HOURS", "24")
	t.Setenv("API_BASE_URL", "http://localhost:8080")
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:3000")
	t.Setenv("AI_SERVICE_URL", "http://localhost:8000")
}

func clearRequired(t *testing.T) {
	t.Helper()
	for _, k := range requiredVars {
		os.Unsetenv(k)
	}
}

func TestLoad_AllVarsPresent(t *testing.T) {
	setAllRequired(t)
	if err := config.Load(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestLoad_MissingVar_ReturnsError(t *testing.T) {
	clearRequired(t)
	setAllRequired(t)
	os.Unsetenv("PORT")

	err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing PORT, got nil")
	}
}

func TestLoad_AllMissingVars_ReturnsError(t *testing.T) {
	clearRequired(t)

	err := config.Load()
	if err == nil {
		t.Fatal("expected error when all vars are missing, got nil")
	}
}

func TestLoad_ShortJWTSecret_ReturnsError(t *testing.T) {
	setAllRequired(t)
	t.Setenv("JWT_SECRET", "short") // fewer than 32 chars

	err := config.Load()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET, got nil")
	}
}

func TestLoad_Exactly32CharJWTSecret_Succeeds(t *testing.T) {
	setAllRequired(t)
	t.Setenv("JWT_SECRET", "exactly-32-chars-long-secret!!!x") // exactly 32

	if err := config.Load(); err != nil {
		t.Errorf("expected no error for 32-char secret, got: %v", err)
	}
}

func TestLoadEnvOrFatal_Succeeds_WhenAllVarsPresent(t *testing.T) {
	setAllRequired(t)
	// If this panics/fatal the test binary will exit — that would be a failure.
	config.LoadEnvOrFatal()
}
