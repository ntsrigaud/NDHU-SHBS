package services

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Database wraps a sqlx.DB connection and exposes a liveness probe used by
// the health-check middleware.
type Database struct {
	DB *sqlx.DB
}

// Ready returns true when the database can be reached. Used by the readiness
// probe — a false result causes the load balancer to stop sending traffic.
func (d *Database) Ready() bool {
	if err := d.DB.Ping(); err != nil {
		log.Printf("Database ping failed: %v", err)
		return false
	}
	return true
}
