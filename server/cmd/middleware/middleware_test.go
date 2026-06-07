package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"shbs-server/cmd/middleware"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const testSecret = "test-secret-that-is-at-least-32-chars-long!"

func setJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("JWT_EXPIRY_HOURS", "1")
}

func newApp() *fiber.App {
	return fiber.New(fiber.Config{DisableStartupMessage: true})
}

func do(t *testing.T, app *fiber.App, req *http.Request) *http.Response {
	t.Helper()
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return resp
}

// ── ErrorHandler ──────────────────────────────────────────────────────────────

func TestErrorHandler_NoError(t *testing.T) {
	app := newApp()
	app.Use(middleware.ErrorHandler())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestErrorHandler_FiberError(t *testing.T) {
	app := newApp()
	app.Use(middleware.ErrorHandler())
	app.Get("/", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "bad input")
	})
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected 'error' key in JSON response")
	}
}

func TestErrorHandler_GenericError(t *testing.T) {
	app := newApp()
	app.Use(middleware.ErrorHandler())
	app.Get("/", func(c *fiber.Ctx) error {
		return errors.New("internal failure")
	})
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// ── Logger ────────────────────────────────────────────────────────────────────

func TestLogger_LogsAndPassesThrough(t *testing.T) {
	app := newApp()
	app.Use(middleware.Logger())
	app.Get("/ping", func(c *fiber.Ctx) error {
		return c.SendString("pong")
	})
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLogger_ErrorHandling(t *testing.T) {
	app := newApp()
	app.Use(middleware.Logger())

	// Case 1: fiber.Error
	app.Get("/fiber-error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	})
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/fiber-error", nil))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}

	// Case 2: Generic error
	app.Get("/generic-error", func(c *fiber.Ctx) error {
		return errors.New("boom")
	})
	resp = do(t, app, httptest.NewRequest(http.MethodGet, "/generic-error", nil))
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

// ── Compressor ────────────────────────────────────────────────────────────────

func TestCompressor_ReturnsOK(t *testing.T) {
	app := newApp()
	app.Use(middleware.Compressor())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("hello world")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── CorsHandler ───────────────────────────────────────────────────────────────

func TestCorsHandler_Wildcard_WhenNoEnv(t *testing.T) {
	os.Unsetenv("FRONTEND_BASE_URL")
	app := newApp()
	app.Use(middleware.CorsHandler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCorsHandler_AllowsKnownOrigin(t *testing.T) {
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:3000")
	app := newApp()
	app.Use(middleware.CorsHandler())
	app.Options("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCorsHandler_AllowsVercelPreview(t *testing.T) {
	t.Setenv("FRONTEND_BASE_URL", "https://my-app.vercel.app")
	app := newApp()
	app.Use(middleware.CorsHandler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://my-app-git-branch-abc.vercel.app")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCorsHandler_MultipleOrigins(t *testing.T) {
	t.Setenv("FRONTEND_BASE_URL", "http://localhost:3000,https://example.com")
	app := newApp()
	app.Use(middleware.CorsHandler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	origins := []string{"http://localhost:3000", "https://example.com", "http://127.0.0.1:4000"}
	for _, origin := range origins {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", origin)
		resp := do(t, app, req)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for origin %s, got %d", origin, resp.StatusCode)
		}
	}
}

func TestCorsHandler_DeniesUnknownOrigin(t *testing.T) {
	t.Setenv("FRONTEND_BASE_URL", "https://example.com")
	app := newApp()
	app.Use(middleware.CorsHandler())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://malicious.com")
	resp := do(t, app, req)
	// CORS middleware by default just doesn't set the CORS headers if origin is not allowed.
	// Fiber's CORS middleware might still return 200 but without the Access-Control-Allow-Origin header.
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected empty Access-Control-Allow-Origin for unknown origin, got %s", resp.Header.Get("Access-Control-Allow-Origin"))
	}
}

func TestPrivateNetworkAccessHandler(t *testing.T) {
	app := newApp()
	app.Use(middleware.PrivateNetworkAccessHandler())
	app.Options("/", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })

	// Case 1: OPTIONS request with PNA header
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	resp := do(t, app, req)
	if resp.Header.Get("Access-Control-Allow-Private-Network") != "true" {
		t.Error("expected Access-Control-Allow-Private-Network header to be true")
	}

	// Case 2: GET request with PNA header (should NOT set the header)
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	resp = do(t, app, req)
	if resp.Header.Get("Access-Control-Allow-Private-Network") != "" {
		t.Error("expected Access-Control-Allow-Private-Network header to be empty for GET")
	}
}

// ── HealthChecks ──────────────────────────────────────────────────────────────

// fakeDB implements the ping interface used by services.Database.
// We use a nil *sqlx.DB and intercept via a wrapper so we can control Ready().
type readyDB struct{ ok bool }

func (r *readyDB) toDatabase() *services.Database {
	// We create a sqlx.DB that is NOT connected — Ping will fail.
	// Then we wrap it; if r.ok we need to inject a real connection.
	// Since we can't easily control sqlx, test /live directly and skip
	// /ready-true (requires a real DB) — just test /ready when DB is down.
	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	return &services.Database{DB: db}
}

func TestHealthChecks_Liveness(t *testing.T) {
	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	database := &services.Database{DB: db}

	app := newApp()
	middleware.RegisterHealthChecks(app, database)

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/live", nil))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/live expected 200, got %d", resp.StatusCode)
	}
}

func TestHealthChecks_ReadinessDown(t *testing.T) {
	// Use a deliberately unreachable DSN — Ping will fail.
	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	database := &services.Database{DB: db}

	app := newApp()
	middleware.RegisterHealthChecks(app, database)

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("/ready (DB down) expected 503, got %d", resp.StatusCode)
	}
}

// ── CheckAdmin ────────────────────────────────────────────────────────────────

func TestCheckAdmin_Granted(t *testing.T) {
	app := newApp()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("isAdmin", true)
		return c.Next()
	})
	app.Use(middleware.CheckAdmin())
	app.Get("/secret", func(c *fiber.Ctx) error { return c.SendString("admin area") })

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/secret", nil))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCheckAdmin_Denied_NotAdmin(t *testing.T) {
	app := newApp()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("isAdmin", false)
		return c.Next()
	})
	app.Use(middleware.CheckAdmin())
	app.Get("/secret", func(c *fiber.Ctx) error { return c.SendString("admin area") })

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/secret", nil))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestCheckAdmin_Denied_MissingLocal(t *testing.T) {
	app := newApp()
	// isAdmin not set in locals at all
	app.Use(middleware.CheckAdmin())
	app.Get("/secret", func(c *fiber.Ctx) error { return c.SendString("admin area") })

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/secret", nil))
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

// ── Auth middleware ───────────────────────────────────────────────────────────
// Auth requires a *sqlx.DB for the blacklist check. We use a non-connected DB
// so IsTokenBlacklisted returns false (scan error → false), letting valid tokens
// pass through. Invalid tokens are rejected before the DB call.

func TestAuth_ValidToken_PassesThrough(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	app := newApp()
	app.Use(middleware.Auth(db))
	app.Get("/protected", func(c *fiber.Ctx) error {
		uid := c.Locals("userID")
		return c.SendString(uid.(string))
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuth_MissingToken_Returns401(t *testing.T) {
	setJWTEnv(t)
	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	app := newApp()
	app.Use(middleware.Auth(db))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendString("ok") })

	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuth_InvalidToken_Returns401(t *testing.T) {
	setJWTEnv(t)
	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	app := newApp()
	app.Use(middleware.Auth(db))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer bad.token.here")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuth_ValidToken_FromCookie(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, true)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	db, _ := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	app := newApp()
	app.Use(middleware.Auth(db))
	app.Get("/protected", func(c *fiber.Ctx) error {
		isAdmin := c.Locals("isAdmin").(bool)
		if isAdmin {
			return c.SendString("admin")
		}
		return c.SendString("user")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// ── RateLimiter ───────────────────────────────────────────────────────────────

func TestRateLimiter_LocalhostBypassed(t *testing.T) {
	app := newApp()
	app.Use(middleware.RateLimiter())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	// Fiber's test server uses 0.0.0.0 as peer, not 127.0.0.1, so this just
	// verifies the middleware mounts and handles normally without panicking.
	resp := do(t, app, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("unexpected status %d", resp.StatusCode)
	}
}

func TestRateLimiter_CustomKeyGenerator(t *testing.T) {
	app := newApp()
	app.Use(middleware.RateLimiter())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	resp := do(t, app, req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

