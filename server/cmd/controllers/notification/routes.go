package notification

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers notification routes under /notifications. All require authentication.
func Mount(router fiber.Router, db *sqlx.DB) {
	g := router.Group("/notifications", middleware.Auth(db))

	g.Get("/", HandleListNotifications(db))
	g.Patch("/:id", HandleMarkAsRead(db))
}
