package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ── Shared ───────────────────────────────────────────────────────────────────

type SwaggerErrorResponse struct {
	Error string `json:"error"`
}

type SwaggerMessageResponse struct {
	Message string `json:"message"`
}

// ── Auth ─────────────────────────────────────────────────────────────────────

type SwaggerLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SwaggerLoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

type SwaggerRegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SwaggerResendVerificationRequest struct {
	Email string `json:"email"`
}

// ── User ─────────────────────────────────────────────────────────────────────

type SwaggerUpdateUserRequest struct {
	Name          *string `json:"name"`
	AvatarImageID *string `json:"avatar_image_id"`
}

// ── Listing ──────────────────────────────────────────────────────────────────

type SwaggerCreateListingRequest struct {
	Title       string          `json:"title"`
	Author      string          `json:"author"`
	ISBN        string          `json:"isbn"`
	CourseCode  string          `json:"course_code"`
	Department  string          `json:"department"`
	Price       decimal.Decimal `json:"price"`
	Condition   string          `json:"condition"`
	Description string          `json:"description"`
	ImageIDs    []uuid.UUID     `json:"image_ids"`
}

type SwaggerUpdateListingRequest struct {
	Title       *string          `json:"title"`
	Author      *string          `json:"author"`
	ISBN        *string          `json:"isbn"`
	CourseCode  *string          `json:"course_code"`
	Department  *string          `json:"department"`
	Price       *decimal.Decimal `json:"price"`
	Condition   *string          `json:"condition"`
	Status      *string          `json:"status"`
	Description *string          `json:"description"`
	ImageIDs    *[]uuid.UUID     `json:"image_ids"`
}

// ── Cart ─────────────────────────────────────────────────────────────────────

type SwaggerAddToCartRequest struct {
	ListingID uuid.UUID `json:"listing_id"`
}

// ── Message ──────────────────────────────────────────────────────────────────

type SwaggerSendMessageRequest struct {
	Body string `json:"body"`
}

// ── Image ────────────────────────────────────────────────────────────────────

type SwaggerRegisterImageRequest struct {
	S3Key  string `json:"s3_key"`
	CdnURL string `json:"cdn_url"`
}

// SwaggerUploadImageResponse is returned by POST /images/upload.
// It contains the assigned image ID and the public CDN URL.
type SwaggerUploadImageResponse struct {
	Image ImageResponse `json:"image"`
}

// ── Response view types (used in Swagger docs) ───────────────────────────────

// ListingWithImages enriches a BookListing with seller info and image URLs.
type ListingWithImages struct {
	ID           uuid.UUID       `db:"id"            json:"id"`
	SellerID     uuid.UUID       `db:"seller_id"     json:"seller_id"`
	SellerName   string          `db:"seller_name"   json:"seller_name"`
	SellerAvatar *string         `db:"seller_avatar" json:"seller_avatar,omitempty"`
	Title        string          `db:"title"         json:"title"`
	Author       string          `db:"author"        json:"author"`
	ISBN         *string         `db:"isbn"          json:"isbn,omitempty"`
	CourseCode   *string         `db:"course_code"   json:"course_code,omitempty"`
	Department   *string         `db:"department"    json:"department,omitempty"`
	Price        decimal.Decimal `db:"price"         json:"price"`
	Condition    string          `db:"condition"     json:"condition"`
	Status       string          `db:"status"        json:"status"`
	Description  *string         `db:"description"   json:"description,omitempty"`
	AIConfidence *float64        `db:"ai_confidence" json:"ai_confidence,omitempty"`
	ImageURLs    []string        `db:"image_urls"    json:"image_urls"`
	CreatedAt    time.Time       `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"    json:"updated_at"`
}

// CartItemResponse represents a single item in the cart.
type CartItemResponse struct {
	ID        uuid.UUID        `db:"id"         json:"id"`
	Listing   ListingWithImages `json:"listing"`
	CreatedAt time.Time        `db:"created_at" json:"added_at"`
}

// OrderItemResponse represents a book within an order.
type OrderItemResponse struct {
	ID              uuid.UUID        `db:"id"                json:"id"`
	PriceAtPurchase decimal.Decimal  `db:"price_at_purchase" json:"price_at_purchase"`
	Listing         ListingWithImages `json:"listing"`
}

// OrderResponse represents a full order with its items.
type OrderResponse struct {
	ID          uuid.UUID           `db:"id"           json:"id"`
	Status      string              `db:"status"       json:"status"`
	TotalAmount decimal.Decimal     `db:"total_amount" json:"total_amount"`
	CreatedAt   time.Time           `db:"created_at"   json:"created_at"`
	Items       []OrderItemResponse `json:"items"`
}

// MessageResponse represents a message with sender info.
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

// ConversationResponse represents a summary of a conversation thread.
type ConversationResponse struct {
	ListingID     uuid.UUID `db:"listing_id"      json:"listing_id"`
	ListingTitle  string    `db:"listing_title"   json:"listing_title"`
	OtherUserID   uuid.UUID `db:"other_user_id"   json:"other_user_id"`
	OtherUserName string    `db:"other_user_name" json:"other_user_name"`
	LastMessage   string    `db:"last_message"    json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
	UnreadCount   int       `db:"unread_count"    json:"unread_count"`
}

// NotificationResponse represents a user notification.
type NotificationResponse struct {
	ID        uuid.UUID       `db:"id"         json:"id"`
	Type      string          `db:"type"       json:"type"`
	Payload   json.RawMessage `db:"payload"    json:"payload"`
	IsRead    bool            `db:"is_read"    json:"is_read"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// ImageResponse is the standard response for image operations.
type ImageResponse struct {
	ID     uuid.UUID `json:"id"`
	CdnURL string    `json:"cdn_url"`
}
