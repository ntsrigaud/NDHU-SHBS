package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotificationResponse represents a user notification.
type NotificationResponse struct {
	ID        uuid.UUID       `db:"id"         json:"id"`
	Type      string          `db:"type"       json:"type"`
	Payload   json.RawMessage `db:"payload"    json:"payload"`
	IsRead    bool            `db:"is_read"    json:"is_read"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}
