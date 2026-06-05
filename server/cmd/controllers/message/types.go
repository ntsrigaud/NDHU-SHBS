package message

import (
	"time"

	"github.com/google/uuid"
)

// SendMessageRequest represents the body for POST /messages.
type SendMessageRequest struct {
	ListingID  uuid.UUID `json:"listing_id"`
	ReceiverID uuid.UUID `json:"receiver_id"`
	Body       string    `json:"body"`
}

// MessageResponse represents a message with sender/receiver info.
type MessageResponse struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	ListingID  uuid.UUID `db:"listing_id"  json:"listing_id"`
	SenderID   uuid.UUID `db:"sender_id"   json:"sender_id"`
	SenderName string    `db:"sender_name" json:"sender_name"`
	ReceiverID uuid.UUID `db:"receiver_id" json:"receiver_id"`
	Body       string    `db:"body"        json:"body"`
	IsRead     bool      `db:"is_read"     json:"is_read"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
}
