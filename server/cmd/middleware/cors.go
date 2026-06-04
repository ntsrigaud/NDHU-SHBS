package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CorsHandler configures CORS.
//
// Allowed origins are built from two sources:
//   - FRONTEND_BASE_URL — comma-separated list of explicit origins (e.g. the
//     production Vercel URL and/or the Tailscale staging IP).
//   - Any origin whose host ends with ".vercel.app" — this covers the unique
//     preview URLs that Vercel generates for every pull-request deployment so
//     developers can test against the real API without manual allow-listing.
//
// If FRONTEND_BASE_URL is empty the handler falls back to a wildcard, which
// is only acceptable in local development (the env validator rejects an empty
// value in production).
func CorsHandler() fiber.Handler {
	raw := strings.TrimSpace(os.Getenv("FRONTEND_BASE_URL"))

	// Development fallback — wildcard, no credentials.
	if raw == "" {
		return cors.New(cors.Config{
			AllowOrigins:     "*",
			AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
			AllowCredentials: false,
		})
	}

	known := strings.Split(raw, ",")

	return cors.New(cors.Config{
		// AllowOriginsFunc is evaluated when AllowOrigins does not match,
		// letting us handle the dynamic *.vercel.app preview domains.
		AllowOrigins: raw,
		AllowOriginsFunc: func(origin string) bool {
			origin = strings.TrimSpace(origin)
			for _, o := range known {
				if strings.TrimSpace(o) == origin {
					return true
				}
			}
			// Allow all Vercel preview deployments (*.vercel.app).
			// For production hardening, replace with a specific project slug check.
			return strings.HasSuffix(origin, ".vercel.app")
		},
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	})
}
