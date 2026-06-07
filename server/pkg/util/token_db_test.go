package util_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"shbs-server/pkg/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// unreachableDB returns a *sqlx.DB that can never connect.
// sqlx.Open succeeds (no actual connection attempt); every DB operation will
// return an error, exercising the error-handling branches of token functions.
func unreachableDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sqlx.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// ── IsTokenBlacklisted ───────────────────────────────────────────────────────

func TestIsTokenBlacklisted_DBError_ReturnsFalse(t *testing.T) {
	db := unreachableDB(t)
	if util.IsTokenBlacklisted(db, "any-token") {
		t.Error("expected false when DB is unreachable")
	}
}

// ── DeleteExpiredTokens ──────────────────────────────────────────────────────

func TestDeleteExpiredTokens_DBError_NoPanic(t *testing.T) {
	db := unreachableDB(t)
	// Should not panic — just logs the error internally.
	util.DeleteExpiredTokens(db)
}

// ── DeleteExpiredVerificationTokens ─────────────────────────────────────────

func TestDeleteExpiredVerificationTokens_DBError_NoPanic(t *testing.T) {
	db := unreachableDB(t)
	// Should not panic — just logs the error internally.
	util.DeleteExpiredVerificationTokens(db)
}

// ── InvalidateToken ──────────────────────────────────────────────────────────

// InvalidateToken requires a *fiber.Ctx to write the error response, so we
// invoke it inside a Fiber handler and test via app.Test.

func TestInvalidateToken_DBError_Returns500(t *testing.T) {
	db := unreachableDB(t)

	app := fiberApp() // declared in jwt_test.go (same package util_test)
	app.Get("/invalidate", func(c *fiber.Ctx) error {
		return util.InvalidateToken(db, c, "test-token", time.Now().Add(time.Hour))
	})

	req := httptest.NewRequest(http.MethodGet, "/invalidate", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500 when DB is unreachable, got %d", resp.StatusCode)
	}
}

// ── CreateVerificationToken ──────────────────────────────────────────────────

func TestCreateVerificationToken_DBError(t *testing.T) {
	db := unreachableDB(t)
	_, err := util.CreateVerificationToken(db, uuid.New(), "verify", time.Hour)
	if err == nil {
		t.Error("expected error when DB is unreachable, got nil")
	}
}
