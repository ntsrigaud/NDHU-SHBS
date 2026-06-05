package order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"shbs-server/cmd/controllers/listing"
)

// OrderItemResponse represents a book within an order.
type OrderItemResponse struct {
	ID              uuid.UUID                 `db:"id"                json:"id"`
	PriceAtPurchase decimal.Decimal           `db:"price_at_purchase" json:"price_at_purchase"`
	Listing         listing.ListingWithImages `json:"listing"`
}

// OrderResponse represents a full order with its items.
type OrderResponse struct {
	ID          uuid.UUID           `db:"id"           json:"id"`
	Status      string              `db:"status"       json:"status"`
	TotalAmount decimal.Decimal     `db:"total_amount" json:"total_amount"`
	CreatedAt   time.Time           `db:"created_at"   json:"created_at"`
	Items       []OrderItemResponse `json:"items"`
}
