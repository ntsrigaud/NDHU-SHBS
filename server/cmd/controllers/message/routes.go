package message

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers message routes under /messages. All require authentication.
func Mount(router fiber.Router, db *sqlx.DB) {
	g := router.Group("/messages", middleware.Auth(db))

	g.Get("/", HandleListMessages(db))
	g.Post("/", HandleSendMessage(db))
}
