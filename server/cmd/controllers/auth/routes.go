package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"

	"shbs-server/cmd/middleware"
	"shbs-server/pkg/services"
)

// Mount registers all /auth routes on the provided router group.
//
//	POST   /auth/register          – create local account + send verification email
//	GET    /auth/verify            – consume email verification token
//	POST   /auth/login             – authenticate and issue JWT
//	POST   /auth/logout            – blacklist token and clear cookie  [auth required]
//	GET    /auth/sso/login         – redirect to NDHU CAS login page
//	GET    /auth/sso/callback      – validate CAS ticket and issue JWT
func Mount(router fiber.Router, db *sqlx.DB, emailSvc *services.EmailService) {
	g := router.Group("/auth")

	g.Post("/register", HandleRegister(db, emailSvc))
	g.Get("/verify", HandleVerify(db))
	g.Post("/login", HandleLogin(db))
	g.Post("/logout", middleware.Auth(db), HandleLogout(db))

	g.Get("/sso/login", HandleSSOLogin())
	g.Get("/sso/callback", HandleSSOCallback(db))
}
