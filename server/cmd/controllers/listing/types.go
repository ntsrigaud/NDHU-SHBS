package listing

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// CreateListingRequest carries all fields needed to publish a new listing.
type CreateListingRequest struct {
	Title       string          `json:"title"`
	Author      string          `json:"author"`
	ISBN        string          `json:"isbn"`
	CourseCode  string          `json:"course_code"`
	Department  string          `json:"department"`
	Price       decimal.Decimal `json:"price"`
	Condition   string          `json:"condition"`
	Description string          `json:"description"`
	ImageIDs    []uuid.UUID     `json:"image_ids"`
}

// UpdateListingRequest is a partial update — only non-nil fields are applied.
type UpdateListingRequest struct {
	Title       *string          `json:"title"`
	Author      *string          `json:"author"`
	ISBN        *string          `json:"isbn"`
	CourseCode  *string          `json:"course_code"`
	Department  *string          `json:"department"`
	Price       *decimal.Decimal `json:"price"`
	Condition   *string          `json:"condition"`
	Status      *string          `json:"status"`
	Description *string          `json:"description"`
	ImageIDs    *[]uuid.UUID     `json:"image_ids"`
}

// ListingsPage is the envelope returned by GET /listings.
type ListingsPage struct {
	Data  []ListingWithImages `json:"data"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// ListingWithImages enriches a BookListing with its image IDs and seller info.
type ListingWithImages struct {
	ID           uuid.UUID       `db:"id"            json:"id"`
	SellerID     uuid.UUID       `db:"seller_id"     json:"seller_id"`
	SellerName   string          `db:"seller_name"   json:"seller_name"`
	SellerAvatar *string         `db:"seller_avatar" json:"seller_avatar,omitempty"`
	Title        string          `db:"title"         json:"title"`
	Author       string          `db:"author"        json:"author"`
	ISBN         *string         `db:"isbn"          json:"isbn,omitempty"`
	CourseCode   *string         `db:"course_code"   json:"course_code,omitempty"`
	Department   *string         `db:"department"    json:"department,omitempty"`
	Price        decimal.Decimal `db:"price"         json:"price"`
	Condition    string          `db:"condition"     json:"condition"`
	Status       string          `db:"status"        json:"status"`
	Description  *string         `db:"description"   json:"description,omitempty"`
	AIConfidence *float64        `db:"ai_confidence" json:"ai_confidence,omitempty"`
	ImageURLs    pq.StringArray  `db:"image_urls"    json:"image_urls"`
	CreatedAt    time.Time       `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"    json:"updated_at"`
}

// validConditions contains the allowed values for Condition.
var validConditions = map[string]bool{
	"good": true, "moderate": true, "poor": true,
}

// sellerEditableStatuses are the status values a seller may set directly.
var sellerEditableStatuses = map[string]bool{
	"active": true, "delisted": true,
}

func normalizeCreateRequest(r *CreateListingRequest) {
	r.Title = strings.TrimSpace(r.Title)
	r.Author = strings.TrimSpace(r.Author)
	r.ISBN = strings.TrimSpace(r.ISBN)
	r.CourseCode = strings.TrimSpace(r.CourseCode)
	r.Department = strings.TrimSpace(r.Department)
	r.Condition = strings.ToLower(strings.TrimSpace(r.Condition))
	r.Description = strings.TrimSpace(r.Description)
}

// validateCreateRequest returns a non-empty string if the request is invalid.
func validateCreateRequest(r *CreateListingRequest) string {
	if r.Title == "" {
		return "title is required"
	}
	if r.Author == "" {
		return "author is required"
	}
	if r.Price.IsNegative() {
		return "price must be >= 0"
	}
	if !validConditions[r.Condition] {
		return "condition must be one of: good, moderate, poor"
	}
	return ""
}
