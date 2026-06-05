package image

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers image routes on the provided router.
func Mount(router fiber.Router, db *sqlx.DB) {
	images := router.Group("/images")

	// Public lookup
	images.Get("/:id", HandleGetImage(db))

	// Auth-protected registration
	protected := images.Group("", middleware.Auth(db))
	protected.Post("/", HandleRegisterImage(db))
}
