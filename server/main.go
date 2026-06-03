package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"shbs-server/cmd/middleware"
	"shbs-server/pkg/config"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// @title           NDHU Second-Hand Book Store API
// @version         1.0
// @description     REST API for the NDHU campus second-hand textbook marketplace.
// @termsOfService  http://swagger.io/terms/
// @contact.name    SHBS Dev Team
// @license.name    MIT
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	config.LoadEnvOrFatal()

	db, err := sqlx.Connect("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}
	defer db.Close()

	database := &services.Database{DB: db}

	// Periodic cleanup of expired tokens (every 6 hours).
	services.SetupScheduler(6*time.Hour, func() {
		util.DeleteExpiredTokens(db)
		util.DeleteExpiredVerificationTokens(db)
	})

	app := fiber.New(fiber.Config{
		AppName:      "SHBS API v1",
		ErrorHandler: nil, // handled by ErrorHandler middleware
	})

	// ── Global middleware ─────────────────────────────────────────────────────
	app.Use(middleware.ErrorHandler())
	app.Use(middleware.CorsHandler())
	app.Use(middleware.Logger())
	app.Use(middleware.Compressor())
	app.Use(middleware.RateLimiter())

	// ── Health probes ─────────────────────────────────────────────────────────
	middleware.RegisterHealthChecks(app, database)

	// ── Swagger UI ────────────────────────────────────────────────────────────
	// The /swagger/* route is registered once the docs package is generated via
	// `make generate-api`. Phase 0 omits it to keep the binary free of the
	// swag-generated dependency.

	// ── API routes ────────────────────────────────────────────────────────────
	api := app.Group("/api/v1")
	registerRoutes(api, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("SHBS API listening on :%s", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}

// registerRoutes attaches all controller groups. Controller packages will be
// added in Phase 1 (auth), Phase 2 (listings), etc. The stubs below keep the
// file compilable during Phase 0.
func registerRoutes(api fiber.Router, db *sqlx.DB) {
	// Phase 0 placeholder — controllers will be wired here in subsequent phases.
	_ = api
	_ = db
}
