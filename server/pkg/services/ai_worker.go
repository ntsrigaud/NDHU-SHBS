package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AIWorker processes listings that are marked as unprocessed.
type AIWorker struct {
	DB *sqlx.DB
	AI *AIService
}

// NewAIWorker creates a new worker instance.
func NewAIWorker(db *sqlx.DB, ai *AIService) *AIWorker {
	return &AIWorker{DB: db, AI: ai}
}

// ProcessPendingListings finds all listings with ai_processed = false and
// runs metadata and condition analysis on them.
func (w *AIWorker) ProcessPendingListings() {
	var listings []struct {
		ID uuid.UUID `db:"id"`
	}

	// Only process listings that haven't been processed yet.
	err := w.DB.Select(&listings, "SELECT id FROM book_listings WHERE ai_processed = FALSE LIMIT 10")
	if err != nil {
		log.Printf("AIWorker: error fetching pending listings: %v", err)
		return
	}

	for _, l := range listings {
		w.ProcessListing(l.ID)
	}
}

// ProcessListing handles a single listing's AI analysis.
func (w *AIWorker) ProcessListing(id uuid.UUID) {
	log.Printf("AIWorker: processing listing %s", id)

	// 1. Get image URLs for this listing.
	var imageURLs []string
	err := w.DB.Select(&imageURLs, `
		SELECT i.cdn_url 
		FROM listing_images li 
		JOIN images i ON li.image_id = i.id 
		WHERE li.listing_id = $1 
		ORDER BY li.display_order`, id)
	if err != nil {
		log.Printf("AIWorker: error fetching images for listing %s: %v", id, err)
		return
	}

	if len(imageURLs) == 0 {
		log.Printf("AIWorker: listing %s has no images, marking as processed anyway", id)
		w.DB.Exec("UPDATE book_listings SET ai_processed = TRUE WHERE id = $1", id)
		return
	}

	// 2. Run Metadata Analysis.
	meta, metaErr := w.AI.AnalyzeMetadata(imageURLs)
	if metaErr != nil {
		log.Printf("AIWorker: metadata analysis failed for %s: %v", id, metaErr)
	}

	// 3. Run Condition Analysis.
	cond, condErr := w.AI.AnalyzeCondition(imageURLs)
	if condErr != nil {
		log.Printf("AIWorker: condition analysis failed for %s: %v", id, condErr)
	}

	// 4. Update listing with results.
	tx, err := w.DB.Beginx()
	if err != nil {
		return
	}
	defer tx.Rollback()

	updateFields := []string{"ai_processed = TRUE", "updated_at = NOW()"}
	updateArgs := []any{id}
	p := 2

	if meta != nil && meta.Confidence > 0 {
		if meta.Title != nil {
			updateFields = append(updateFields, fmt.Sprintf("title = $%d", p))
			updateArgs = append(updateArgs, *meta.Title)
			p++
		}
		if meta.Author != nil {
			updateFields = append(updateFields, fmt.Sprintf("author = $%d", p))
			updateArgs = append(updateArgs, *meta.Author)
			p++
		}
		if meta.ISBN != nil {
			updateFields = append(updateFields, fmt.Sprintf("isbn = $%d", p))
			updateArgs = append(updateArgs, *meta.ISBN)
			p++
		}
		updateFields = append(updateFields, fmt.Sprintf("ai_confidence = $%d", p))
		updateArgs = append(updateArgs, meta.Confidence)
		p++
	}

	if cond != nil {
		updateFields = append(updateFields, fmt.Sprintf("condition_score = $%d", p))
		updateArgs = append(updateArgs, cond.Score)
		p++
	}

	finalQuery := "UPDATE book_listings SET " + strings.Join(updateFields, ", ") + " WHERE id = $1"
	_, err = tx.Exec(finalQuery, updateArgs...)
	if err != nil {
		log.Printf("AIWorker: error updating listing %s: %v", id, err)
		return
	}

	// 5. Create notification for seller.
	var sellerID uuid.UUID
	w.DB.Get(&sellerID, "SELECT seller_id FROM book_listings WHERE id = $1", id)

	payload, _ := json.Marshal(map[string]any{
		"listing_id": id,
		"message":    "AI has finished processing your book listing. Please review the extracted metadata.",
	})

	_, err = tx.Exec(`
		INSERT INTO notifications (user_id, type, payload)
		VALUES ($1, 'listing_ai_ready', $2)`,
		sellerID, payload,
	)
	if err != nil {
		log.Printf("AIWorker: error creating notification for %s: %v", id, err)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("AIWorker: error committing transaction for %s: %v", id, err)
	}
}
