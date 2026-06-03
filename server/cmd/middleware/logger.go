package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger records the IP, method, path, response status, and elapsed time for
// every request. Uses the standard library log package to keep the dependency
// count low; swap for zerolog/zap when structured logging is required.
func Logger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Printf("[%s] %s %s → %d (%s)",
			c.IP(),
			c.Method(),
			c.OriginalURL(),
			c.Response().StatusCode(),
			time.Since(start),
		)
		return err
	}
}
