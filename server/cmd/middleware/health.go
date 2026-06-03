package middleware

import (
	"shbs-server/pkg/services"

	"github.com/gofiber/fiber/v2"
)

// RegisterHealthChecks attaches the root-level liveness and readiness probes.
func RegisterHealthChecks(app fiber.Router, database *services.Database) {
	app.Get("/live", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	app.Get("/ready", func(c *fiber.Ctx) error {
		if database.Ready() {
			return c.SendStatus(fiber.StatusOK)
		}

		return c.SendStatus(fiber.StatusServiceUnavailable)
	})
}
