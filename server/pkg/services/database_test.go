package services_test

import (
	"os"
	"testing"

	"shbs-server/pkg/services"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestDatabase_Ready_ReturnsFalse_WhenUnreachable(t *testing.T) {
	db, err := sqlx.Open("postgres", "host=127.0.0.1 port=1 dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("sqlx.Open: %v", err)
	}
	defer db.Close()

	d := &services.Database{DB: db}
	if d.Ready() {
		t.Error("expected Ready() to return false for an unreachable database")
	}
}

// TestDatabase_Ready_ReturnsTrue requires a real PostgreSQL instance reachable
// via DATABASE_URL. It is skipped locally when DATABASE_URL is not set.
func TestDatabase_Ready_ReturnsTrue_WhenConnected(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping live-DB test")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("cannot connect to database (%v) — skipping live-DB test", err)
	}
	defer db.Close()

	d := &services.Database{DB: db}
	if !d.Ready() {
		t.Error("expected Ready() to return true for a reachable database")
	}
}
