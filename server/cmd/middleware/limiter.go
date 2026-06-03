package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimiter applies a per-IP request rate limit to all routes.
// The limits below are intentionally permissive for a prototype; tighten
// per-route in Phase 2 (auth endpoints get stricter limits).
func RateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		// Bypass localhost (e.g. internal health checks, Docker bridge).
		Next: func(c *fiber.Ctx) bool {
			return c.IP() == "127.0.0.1"
		},
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Prefer the real client IP when behind a proxy.
			if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
				return forwarded
			}
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			log.Printf("Rate limit reached: %s %s %s", c.IP(), c.Method(), c.OriginalURL())
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "too many requests — please slow down",
			})
		},
	})
}
