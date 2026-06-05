package cart

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers cart routes under /cart. All require authentication.
func Mount(router fiber.Router, db *sqlx.DB) {
	g := router.Group("/cart", middleware.Auth(db))

	g.Get("/", HandleListCart(db))
	g.Post("/", HandleAddToCart(db))
	g.Delete("/:id", HandleRemoveFromCart(db))
}
