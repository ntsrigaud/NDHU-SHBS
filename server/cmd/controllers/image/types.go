package image

import (
	"github.com/google/uuid"
)

// RegisterImageRequest represents the metadata for an image already uploaded to storage.
// In a real S3 flow, the client uploads directly to S3 (pre-signed URL) and then
// registers the record here.
type RegisterImageRequest struct {
	S3Key  string `json:"s3_key"`
	CdnURL string `json:"cdn_url"`
}

// ImageResponse is the standard response for image operations.
type ImageResponse struct {
	ID     uuid.UUID `json:"id"`
	CdnURL string    `json:"cdn_url"`
}
