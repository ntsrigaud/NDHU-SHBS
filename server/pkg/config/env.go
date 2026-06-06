package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// requiredEnv lists every environment variable the server cannot start without.
var requiredEnv = []string{
	"PORT",
	"DATABASE_URL",
	"POSTGRES_USER",
	"POSTGRES_PASSWORD",
	"POSTGRES_DB",
	"JWT_SECRET",
	"JWT_EXPIRY_HOURS",
	"API_BASE_URL",
	"FRONTEND_BASE_URL",
	"AI_SERVICE_URL",
}

// Load reads the .env file (if present) and validates that all required
// variables are set. It returns a descriptive error on any failure so callers
// can decide how to handle it (e.g. log.Fatal in main, return in tests).
func Load() error {
	// godotenv.Load is best-effort: a missing file is acceptable in environments
	// where variables are injected directly (Docker, CI, cloud runtimes).
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — expecting environment variables to be set externally")
	}

	var missing []string
	for _, key := range requiredEnv {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	return nil
}

// LoadEnvOrFatal calls Load and terminates the process on any error.
// Used by main() where a misconfigured deployment must fail fast.
func LoadEnvOrFatal() {
	if err := Load(); err != nil {
		log.Fatal(err)
	}
}
