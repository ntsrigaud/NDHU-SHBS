package listing

import (
	"strings"

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
}

// UpdateListingRequest is a partial update — only non-nil fields are applied.
// The seller may update any field; the status field allows transitioning to
// "delisted" (taking the listing down) but not directly to "sold" or "reserved"
// (those are managed by the order/cart flow).
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
}

// ListingsPage is the envelope returned by GET /listings.
type ListingsPage struct {
	Data  []ListingWithImages `json:"data"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// ListingWithImages enriches a BookListing with its image IDs in display order.
type ListingWithImages struct {
	ID           interface{}   `json:"id"`
	SellerID     interface{}   `json:"seller_id"`
	Title        string        `json:"title"`
	Author       string        `json:"author"`
	ISBN         interface{}   `json:"isbn,omitempty"`
	CourseCode   interface{}   `json:"course_code,omitempty"`
	Department   interface{}   `json:"department,omitempty"`
	Price        interface{}   `json:"price"`
	Condition    string        `json:"condition"`
	Status       string        `json:"status"`
	Description  interface{}   `json:"description,omitempty"`
	AIConfidence interface{}   `json:"ai_confidence,omitempty"`
	ImageIDs     []interface{} `json:"image_ids"`
	CreatedAt    interface{}   `json:"created_at"`
	UpdatedAt    interface{}   `json:"updated_at"`
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
