package config

import (
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

// LoadEnvOrFatal loads the .env file from the working directory (if present)
// and then verifies that all required variables are set. It calls log.Fatal
// on any failure, intentionally terminating the process at boot time so that
// misconfigured deployments are immediately visible.
func LoadEnvOrFatal() {
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
		log.Fatalf("Missing required environment variables: %s", strings.Join(missing, ", "))
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters long")
	}
}
