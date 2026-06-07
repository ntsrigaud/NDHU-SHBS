package middleware

import (
	"net/url"
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
	allowLoopback := false
	for _, candidate := range known {
		origin := strings.TrimSpace(candidate)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err == nil {
			host := strings.ToLower(parsed.Hostname())
			if host == "localhost" || host == "127.0.0.1" {
				allowLoopback = true
				break
			}
		}
	}

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
			if allowLoopback {
				parsed, err := url.Parse(origin)
				if err == nil {
					host := strings.ToLower(parsed.Hostname())
					if host == "localhost" || host == "127.0.0.1" {
						return true
					}
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

// PrivateNetworkAccessHandler handles the W3C Private Network Access (PNA)
// preflight. Browsers send "Access-Control-Request-Private-Network: true" when
// a public origin (e.g. ndhu-shbs.vercel.app) makes a request whose resolved
// IP falls in a private or CGNAT range (RFC 1918 or 100.64.0.0/10 — the range
// Tailscale uses). The server must echo "Access-Control-Allow-Private-Network:
// true" in the preflight response for the browser to permit the actual request.
//
// This handler must be registered AFTER CorsHandler so that the CORS headers
// are already set before we append the PNA header.
func PrivateNetworkAccessHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions &&
			c.Get("Access-Control-Request-Private-Network") == "true" {
			c.Set("Access-Control-Allow-Private-Network", "true")
		}
		return c.Next()
	}
}
