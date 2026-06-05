package listing

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
)

// Mount registers listing routes on the provided router.
//
// Public:
//
//	GET  /listings       – paginated list with optional filters
//	GET  /listings/:id   – single listing detail
//
// Auth-protected (seller or admin):
//
//	POST   /listings       – create a new listing
//	PUT    /listings/:id   – update own listing
//	DELETE /listings/:id   – soft-delete (delist) own listing; admin may delist any
func Mount(router fiber.Router, db *sqlx.DB) {
	listings := router.Group("/listings")

	// Public routes.
	listings.Get("/", HandleListListings(db))
	listings.Get("/:id", HandleGetListing(db))

	// Auth-protected routes.
	protected := listings.Group("", middleware.Auth(db))
	protected.Post("/", HandleCreateListing(db))
	protected.Put("/:id", HandleUpdateListing(db))
	protected.Delete("/:id", HandleDeleteListing(db))
}
