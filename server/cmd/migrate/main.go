package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// migrate applies every SQL file in the migrations/ directory in lexicographic
// order. Each file is executed inside a transaction; a failure rolls back only
// that file, leaving previously applied migrations intact.
//
// The schema_migrations table acts as a simple applied-file registry. Files
// already recorded there are skipped, making the tool idempotent.
func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("Cannot connect to database: %v", err)
	}
	defer db.Close()

	// Ensure the bookkeeping table exists before we do anything else.
	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("Cannot create schema_migrations: %v", err)
	}

	// Discover migration files.
	migrationsDir := "migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("Cannot read %s: %v", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	applied := 0
	for _, name := range files {
		if alreadyApplied(db, name) {
			log.Printf("Skipping %s (already applied)", name)
			continue
		}
		if err := applyMigration(db, migrationsDir, name); err != nil {
			log.Fatalf("Failed to apply %s: %v", name, err)
		}
		applied++
	}

	if applied == 0 {
		log.Println("Nothing to migrate.")
	} else {
		log.Printf("Successfully applied %d migration(s).", applied)
	}
}

func ensureMigrationsTable(db *sqlx.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	return err
}

func alreadyApplied(db *sqlx.DB, name string) bool {
	var exists bool
	_ = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&exists)
	return exists
}

func applyMigration(db *sqlx.DB, dir, name string) error {
	content, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if _, err := tx.Exec(string(content)); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("execute SQL: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	log.Printf("Applied %s", name)
	return nil
}
