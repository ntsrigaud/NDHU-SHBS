package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// StringSlice scans PostgreSQL text[] values into a JSON-friendly Go slice.
// It is used for response view models where Swagger should still expose an
// array of strings rather than a pq-specific implementation detail.
type StringSlice []string

func (s *StringSlice) Scan(src any) error {
	return pq.Array((*[]string)(s)).Scan(src)
}

// Listing condition values — mirror the DB CHECK constraint.
const (
	ConditionGood     = "good"
	ConditionModerate = "moderate"
	ConditionPoor     = "poor"
)

// Listing status values — mirror the DB CHECK constraint.
const (
	ListingStatusActive   = "active"
	ListingStatusPending  = "pending"
	ListingStatusReserved = "reserved"
	ListingStatusSold     = "sold"
	ListingStatusDelisted = "delisted"
)

// BookListing represents a row in the `book_listings` table.
// Nullable fields (ISBN, course_code, etc.) are pointers so sqlx can scan SQL
// NULLs cleanly without a separate NullString wrapper.
type BookListing struct {
	ID             uuid.UUID       `db:"id"              json:"id"`
	SellerID       uuid.UUID       `db:"seller_id"       json:"seller_id"`
	Title          string          `db:"title"           json:"title"`
	Author         string          `db:"author"          json:"author"`
	ISBN           *string         `db:"isbn"            json:"isbn,omitempty"`
	CourseCode     *string         `db:"course_code"     json:"course_code,omitempty"`
	Department     *string         `db:"department"      json:"department,omitempty"`
	Price          decimal.Decimal `db:"price"           json:"price"`
	Condition      string          `db:"condition"       json:"condition"`
	Status         string          `db:"status"          json:"status"`
	Description    *string         `db:"description"     json:"description,omitempty"`
	AIConfidence   *float64        `db:"ai_confidence"   json:"ai_confidence,omitempty"`
	ConditionScore *float64        `db:"condition_score" json:"condition_score,omitempty"`
	AIProcessed    bool            `db:"ai_processed"    json:"ai_processed"`
	CreatedAt      time.Time       `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"      json:"updated_at"`
}

// ListingImage is a join-table row linking a listing to one of its uploaded images.
// display_order controls the order photos are shown (0 = cover image).
type ListingImage struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	ListingID    uuid.UUID `db:"listing_id"    json:"listing_id"`
	ImageID      uuid.UUID `db:"image_id"      json:"image_id"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
}
