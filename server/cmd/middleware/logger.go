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
		statusCode := c.Response().StatusCode()
		if err != nil {
			if fiberErr, ok := err.(*fiber.Error); ok {
				statusCode = fiberErr.Code
			} else if statusCode < fiber.StatusBadRequest {
				statusCode = fiber.StatusInternalServerError
			}
		}
		log.Printf("[%s] %s %s → %d (%s)",
			c.IP(),
			c.Method(),
			c.OriginalURL(),
			statusCode,
			time.Since(start),
		)
		return err
	}
}
