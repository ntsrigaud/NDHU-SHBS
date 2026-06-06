package model

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a row in the `messages` table.
// Messages are scoped to a listing — each listing has its own buyer↔seller thread.
// receiver_id is explicitly stored (not derived) to make the unread-inbox query
// a single indexed lookup: WHERE receiver_id = $1 AND is_read = FALSE.
type Message struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	ListingID  uuid.UUID `db:"listing_id"  json:"listing_id"`
	SenderID   uuid.UUID `db:"sender_id"   json:"sender_id"`
	ReceiverID uuid.UUID `db:"receiver_id" json:"receiver_id"`
	Body       string    `db:"body"        json:"body"`
	IsRead     bool      `db:"is_read"     json:"is_read"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}
