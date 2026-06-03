package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CorsHandler configures CORS. The allowed origin is read from FRONTEND_BASE_URL;
// if that variable is empty the handler falls back to a wildcard (development
// only — wildcard is rejected in production by the env validator).
func CorsHandler() fiber.Handler {
	allowOrigins := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL"))
	if allowOrigins == "" {
		allowOrigins = "*"
	}

	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: allowOrigins != "*",
	})
}
