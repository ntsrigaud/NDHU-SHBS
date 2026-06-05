package cart

import (
	"time"

	"github.com/google/uuid"
	"shbs-server/cmd/controllers/listing"
)

// AddToCartRequest represents the body for POST /cart.
type AddToCartRequest struct {
	ListingID uuid.UUID `json:"listing_id"`
}

// CartItemResponse represents a single item in the cart.
type CartItemResponse struct {
	ID        uuid.UUID                 `db:"id"         json:"id"`
	Listing   listing.ListingWithImages `db:"listing"    json:"listing"`
	CreatedAt time.Time                 `db:"created_at" json:"added_at"`
}
