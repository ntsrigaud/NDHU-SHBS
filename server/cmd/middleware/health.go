package middleware

import (
	"shbs-server/pkg/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
)

// HealthCheck registers two probes:
//   - GET /live  — always returns 200 (process is alive)
//   - GET /ready — returns 200 only when the database is reachable
func HealthCheck(database *services.Database) fiber.Handler {
	return healthcheck.New(healthcheck.Config{
		LivenessProbe: func(c *fiber.Ctx) bool {
			return true
		},
		LivenessEndpoint: "/live",
		ReadinessProbe: func(c *fiber.Ctx) bool {
			return database.Ready()
		},
		ReadinessEndpoint: "/ready",
	})
}
