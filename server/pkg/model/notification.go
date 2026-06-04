package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Notification type values.
const (
	NotifTypeNewMessage    = "new_message"
	NotifTypeOrderConfirmed = "order_confirmed"
	NotifTypeListingSold   = "listing_sold"
)

// Notification represents a row in the `notifications` table.
// Payload is a JSONB column whose shape varies by type — stored as
// json.RawMessage so the Go layer never has to re-marshal it.
type Notification struct {
	ID        uuid.UUID       `db:"id"         json:"id"`
	UserID    uuid.UUID       `db:"user_id"    json:"user_id"`
	Type      string          `db:"type"       json:"type"`
	Payload   json.RawMessage `db:"payload"    json:"payload"`
	IsRead    bool            `db:"is_read"    json:"is_read"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}
