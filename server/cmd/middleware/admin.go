package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// CheckAdmin asserts that the authenticated user has the admin flag set.
// Must be composed after Auth middleware so that "isAdmin" is already in locals.
func CheckAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		isAdmin, ok := c.Locals("isAdmin").(bool)
		if !ok || !isAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "admin access required",
			})
		}
		return c.Next()
	}
}
