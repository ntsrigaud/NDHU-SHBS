package model

import (
	"time"

	"github.com/google/uuid"
)

// CartItem represents a row in the `cart_items` table.
// The UNIQUE(buyer_id, listing_id) constraint in the schema prevents duplicates,
// so the application never needs to check for them — the DB enforces it.
type CartItem struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	BuyerID   uuid.UUID `db:"buyer_id"   json:"buyer_id"`
	ListingID uuid.UUID `db:"listing_id" json:"listing_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
