package user

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers authenticated user profile routes under /users.
func Mount(router fiber.Router, db *sqlx.DB) {
	g := router.Group("/users", middleware.Auth(db))
	g.Get("/me", HandleGetMe(db))
	g.Put("/me", HandleUpdateMe(db))
}
