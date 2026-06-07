package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestAIWorker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Start Postgres container
	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("shbs_test_worker"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("connect to test db: %v", err)
	}
	defer db.Close()

	// 2. Apply migrations
	applyTestMigrations(t, db)

	// 3. Mock AI Service
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/analyze/metadata" {
			title := "AI Title"
			author := "AI Author"
			isbn := "9781234567890"
			json.NewEncoder(w).Encode(MetadataResult{
				Title:      &title,
				Author:     &author,
				ISBN:       &isbn,
				Confidence: 0.9,
			})
		} else if r.URL.Path == "/analyze/condition" {
			json.NewEncoder(w).Encode(ConditionResult{
				Condition:  "good",
				Score:      0.95,
				Confidence: 0.9,
			})
		}
	}))
	defer aiServer.Close()

	aiSvc := NewAIService(aiServer.URL)
	worker := NewAIWorker(db, aiSvc)

	// 4. Setup test data
	sellerID := uuid.New()
	db.MustExec(`INSERT INTO users (id, name, email) VALUES ($1, 'Seller', 'seller@example.com')`, sellerID)

	imageID := uuid.New()
	db.MustExec(`INSERT INTO images (id, s3_key, cdn_url) VALUES ($1, 'test.jpg', 'http://cdn/test.jpg')`, imageID)

	listingID := uuid.New()
	db.MustExec(`
		INSERT INTO book_listings (id, seller_id, title, author, price, condition, status, ai_processed)
		VALUES ($1, $2, 'Old Title', 'Old Author', 100, 'moderate', 'pending', FALSE)`,
		listingID, sellerID,
	)
	db.MustExec(`INSERT INTO listing_images (listing_id, image_id, display_order) VALUES ($1, $2, 0)`, listingID, imageID)

	// 5. Test ProcessPendingListings
	t.Run("ProcessPendingListings", func(t *testing.T) {
		worker.ProcessPendingListings()

		var l struct {
			Title       string   `db:"title"`
			Author      string   `db:"author"`
			ISBN        *string  `db:"isbn"`
			AIProcessed bool     `db:"ai_processed"`
			Confidence  *float64 `db:"ai_confidence"`
		}
		err := db.Get(&l, "SELECT title, author, isbn, ai_processed, ai_confidence FROM book_listings WHERE id = $1", listingID)
		if err != nil {
			t.Fatalf("failed to fetch listing: %v", err)
		}

		if l.Title != "AI Title" {
			t.Errorf("expected title AI Title, got %s", l.Title)
		}
		if !l.AIProcessed {
			t.Error("expected ai_processed to be true")
		}
		if l.Confidence == nil || *l.Confidence != 0.9 {
			t.Errorf("expected confidence 0.9, got %v", l.Confidence)
		}

		// Check notification
		var count int
		db.Get(&count, "SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND type = 'listing_ai_ready'", sellerID)
		if count != 1 {
			t.Errorf("expected 1 notification, got %d", count)
		}
	})

	t.Run("ProcessListing with no images", func(t *testing.T) {
		noImgListingID := uuid.New()
		db.MustExec(`
			INSERT INTO book_listings (id, seller_id, title, author, price, condition, status, ai_processed)
			VALUES ($1, $2, 'No Img', 'Author', 10, 'good', 'pending', FALSE)`,
			noImgListingID, sellerID,
		)

		worker.ProcessListing(noImgListingID)

		var aiProcessed bool
		db.Get(&aiProcessed, "SELECT ai_processed FROM book_listings WHERE id = $1", noImgListingID)
		if !aiProcessed {
			t.Error("expected ai_processed to be true even for listing with no images")
		}
	})
}

func applyTestMigrations(t *testing.T, db *sqlx.DB) {
	// Simple migration applier for tests
	migrationsDir := "../../migrations"
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		sql, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
}
