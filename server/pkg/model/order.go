package model

import (
	"time"

	"github.com/google/uuid"
)

// Order status values — mirror the DB CHECK constraint.
const (
	OrderStatusPending   = "pending"
	OrderStatusConfirmed = "confirmed"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
)

// Order represents a row in the `orders` table.
// An order is created from a buyer's cart. TotalAmount is the sum of
// PriceAtPurchase for all associated OrderItems — denormalised for fast reads.
type Order struct {
	ID          uuid.UUID `db:"id"           json:"id"`
	BuyerID     uuid.UUID `db:"buyer_id"     json:"buyer_id"`
	Status      string    `db:"status"       json:"status"`
	TotalAmount float64   `db:"total_amount" json:"total_amount"`
	CreatedAt   time.Time `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"   json:"updated_at"`
}

// OrderItem represents a row in the `order_items` table.
// PriceAtPurchase captures the listing price at checkout time so historical
// orders remain accurate even if the seller later changes the price.
type OrderItem struct {
	ID              uuid.UUID `db:"id"               json:"id"`
	OrderID         uuid.UUID `db:"order_id"         json:"order_id"`
	ListingID       uuid.UUID `db:"listing_id"       json:"listing_id"`
	PriceAtPurchase float64   `db:"price_at_purchase" json:"price_at_purchase"`
}
