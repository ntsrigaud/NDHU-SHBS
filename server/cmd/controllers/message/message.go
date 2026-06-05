package message

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// HandleSendMessage handles POST /messages.
func HandleSendMessage(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var req SendMessageRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		req.Body = strings.TrimSpace(req.Body)
		if req.Body == "" {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "message body cannot be empty")
		}
		if len(req.Body) > 2000 {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "message too long (max 2000 chars)")
		}

		// If receiver_id is missing, assume it's the seller of the listing.
		if req.ReceiverID == uuid.Nil {
			var sellerID uuid.UUID
			err := db.Get(&sellerID, `SELECT seller_id FROM book_listings WHERE id = $1`, req.ListingID)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "listing not found")
			}
			req.ReceiverID = sellerID
		}

		if req.ReceiverID == userID {
			return fiber.NewError(fiber.StatusBadRequest, "cannot send message to yourself")
		}

		var msg model.Message
		err = db.QueryRowx(`
			INSERT INTO messages (id, listing_id, sender_id, receiver_id, body)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING *`,
			uuid.New(), req.ListingID, userID, req.ReceiverID, req.Body,
		).StructScan(&msg)

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not send message")
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": msg})
	}
}

// HandleListMessages handles GET /messages?listing_id=...&other_user_id=...
// Returns the conversation between the current user and another user regarding a listing.
func HandleListMessages(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		listingID, err := uuid.Parse(c.Query("listing_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "listing_id is required")
		}

		otherUserID, _ := uuid.Parse(c.Query("other_user_id"))
		// If other_user_id is missing, and the current user is NOT the seller,
		// we assume they want to talk to the seller.
		if otherUserID == uuid.Nil {
			var sellerID uuid.UUID
			err := db.Get(&sellerID, `SELECT seller_id FROM book_listings WHERE id = $1`, listingID)
			if err != nil {
				return fiber.NewError(fiber.StatusNotFound, "listing not found")
			}
			if sellerID != userID {
				otherUserID = sellerID
			} else {
				return fiber.NewError(fiber.StatusBadRequest, "other_user_id is required for sellers")
			}
		}

		var messages []MessageResponse
		query := `
			SELECT m.*, u.name as sender_name
			FROM messages m
			JOIN users u ON m.sender_id = u.id
			WHERE m.listing_id = $1 
			  AND ((m.sender_id = $2 AND m.receiver_id = $3) OR (m.sender_id = $3 AND m.receiver_id = $2))
			ORDER BY m.created_at ASC`

		if err := db.Select(&messages, query, listingID, userID, otherUserID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		// Mark messages as read
		_, _ = db.Exec(`
			UPDATE messages SET is_read = TRUE 
			WHERE listing_id = $1 AND receiver_id = $2 AND sender_id = $3`,
			listingID, userID, otherUserID,
		)

		return c.JSON(fiber.Map{"messages": messages})
	}
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
