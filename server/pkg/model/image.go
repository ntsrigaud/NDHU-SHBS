package model

import (
	"time"

	"github.com/google/uuid"
)

// Image represents a row in the `images` table — the S3/CloudFront asset registry.
// Every uploaded file is registered here before being referenced by other tables.
type Image struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	S3Key     string    `db:"s3_key"     json:"-"`       // internal storage key — never expose to clients
	CdnURL    string    `db:"cdn_url"    json:"cdn_url"` // public CloudFront URL served to clients
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
