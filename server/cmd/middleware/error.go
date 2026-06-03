package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// ErrorHandler converts unhandled errors into a JSON response with an
// appropriate HTTP status code. It is registered as a middleware (not as
// Fiber's app-level ErrorHandler) so it participates in the middleware chain
// and has access to the request context.
func ErrorHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}
