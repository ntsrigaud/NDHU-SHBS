package order

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers order routes under /orders. All require authentication.
func Mount(router fiber.Router, db *sqlx.DB) {
	g := router.Group("/orders", middleware.Auth(db))

	g.Get("/", HandleListOrders(db))
	g.Post("/", HandleCheckout(db))
}
