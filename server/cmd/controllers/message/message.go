package message

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// GetMessages handles GET /listings/:listingId/messages.
//
// @Summary         List messages for a listing
// @Description     Returns the conversation between the authenticated user and the other party for a given listing. Marks received messages as read.
// @Tags            Messages
// @Produce         json
// @Param           listingId path string true "Listing ID"
// @Success         200 {array} model.MessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getMessages
// @Security        BearerAuth
// @Router          /listings/{listingId}/messages [get]
func GetMessages(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
	}

	var sellerID uuid.UUID
	if err := db.Get(&sellerID, `SELECT seller_id FROM book_listings WHERE id = $1`, listingID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "listing not found")
	}

	otherUserID := sellerID
	if userID == sellerID {
		raw := c.Query("other_user_id")
		if raw == "" {
			return fiber.NewError(fiber.StatusBadRequest, "other_user_id is required for the seller")
		}
		otherUserID, err = uuid.Parse(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid other_user_id")
		}
	}

	var messages []model.MessageResponse
	if err := db.Select(&messages, `
		SELECT m.id, m.listing_id, m.sender_id, m.receiver_id,
		       m.body, m.is_read, m.created_at,
		       u.name AS sender_name
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE m.listing_id = $1
		  AND ((m.sender_id = $2 AND m.receiver_id = $3)
		    OR (m.sender_id = $3 AND m.receiver_id = $2))
		ORDER BY m.created_at ASC`,
		listingID, userID, otherUserID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	_, _ = db.Exec(`
		UPDATE messages SET is_read = TRUE
		WHERE listing_id = $1 AND receiver_id = $2 AND sender_id = $3`,
		listingID, userID, otherUserID,
	)

	if messages == nil {
		messages = []model.MessageResponse{}
	}
	return c.JSON(messages)
}

// SendMessage handles POST /listings/:listingId/messages.
//
// @Summary         Send a message about a listing
// @Description     Sends a message from the authenticated user. If the sender is the seller, they must provide the receiver_id (the buyer). If the sender is a buyer, the message is sent to the seller.
// @Tags            Messages
// @Accept          json
// @Produce         json
// @Param           listingId path string true "Listing ID"
// @Param           body body model.SwaggerSendMessageRequest true "Message body and optional receiver_id"
// @Success         201 {object} model.MessageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              sendMessage
// @Security        BearerAuth
// @Router          /listings/{listingId}/messages [post]
func SendMessage(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	listingID, err := uuid.Parse(c.Params("listingId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid listing id")
	}

	var req model.SwaggerSendMessageRequest
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

	var listingInfo struct {
		Title    string    `db:"title"`
		SellerID uuid.UUID `db:"seller_id"`
	}
	if err := db.Get(&listingInfo, `SELECT title, seller_id FROM book_listings WHERE id = $1`, listingID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "listing not found")
	}

	receiverID := listingInfo.SellerID
	if userID == listingInfo.SellerID {
		if req.ReceiverID == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "receiver_id is required for the seller")
		}
		receiverID = req.ReceiverID

		// Verify that there is at least one message from the buyer to this seller for this listing
		// to prevent sellers from spamming random users.
		var exists bool
		err := db.Get(&exists, `
			SELECT EXISTS(
				SELECT 1 FROM messages
				WHERE listing_id = $1 AND sender_id = $2 AND receiver_id = $3
			)`, listingID, receiverID, userID)
		if err != nil || !exists {
			return fiber.NewError(fiber.StatusBadRequest, "sellers can only reply to existing conversations")
		}
	} else {
		// If user is not the seller, they are the buyer sending to the seller.
		// Ensure they don't try to send to someone else via req.ReceiverID (optional: just ignore it)
		receiverID = listingInfo.SellerID
	}

	var msg model.Message
	if err := db.QueryRowx(`
		INSERT INTO messages (id, listing_id, sender_id, receiver_id, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING *`,
		uuid.New(), listingID, userID, receiverID, req.Body,
	).StructScan(&msg); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "could not send message")
	}

	payload := fmt.Sprintf(`{"listing_id":%q,"listing_title":%q,"sender_id":%q,"body_preview":%q}`,
		msg.ListingID, listingInfo.Title, msg.SenderID, truncate(msg.Body, 40))
	_, _ = db.Exec(`
		INSERT INTO notifications (id, user_id, type, payload)
		VALUES ($1, $2, $3, $4)`,
		uuid.New(), msg.ReceiverID, model.NotifTypeNewMessage, payload,
	)

	return c.Status(fiber.StatusCreated).JSON(msg)
}

// GetUnreadMessageCount handles GET /messages/unread-count.
//
// @Summary         Unread message count
// @Description     Returns the number of unread messages for the authenticated user.
// @Tags            Messages
// @Produce         json
// @Success         200 {object} map[string]int
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getUnreadMessageCount
// @Security        BearerAuth
// @Router          /messages/unread-count [get]
func GetUnreadMessageCount(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var count int
	if err := db.Get(&count, `
		SELECT COUNT(*) FROM messages
		WHERE receiver_id = $1 AND is_read = FALSE`, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.JSON(fiber.Map{"count": count})
}

// GetConversations handles GET /messages/conversations.
//
// @Summary         List conversation threads
// @Description     Returns one summary entry per unique (listing, other_user) conversation pair the authenticated user is part of.
// @Tags            Messages
// @Produce         json
// @Success         200 {array} model.ConversationResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getConversations
// @Security        BearerAuth
// @Router          /messages/conversations [get]
func GetConversations(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var conversations []model.ConversationResponse
	if err := db.Select(&conversations, `
		SELECT DISTINCT ON (m.listing_id, other_user.id)
		    m.listing_id,
		    bl.title                                                    AS listing_title,
		    other_user.id                                               AS other_user_id,
		    other_user.name                                             AS other_user_name,
		    m.body                                                      AS last_message,
		    m.created_at                                                AS last_message_at,
		    COUNT(*) FILTER (WHERE m.receiver_id = $1 AND m.is_read = FALSE)
		        OVER (PARTITION BY m.listing_id, other_user.id)        AS unread_count
		FROM messages m
		JOIN book_listings bl ON m.listing_id = bl.id
		JOIN users other_user ON other_user.id = CASE
		    WHEN m.sender_id = $1 THEN m.receiver_id
		    ELSE m.sender_id
		END
		WHERE m.sender_id = $1 OR m.receiver_id = $1
		ORDER BY m.listing_id, other_user.id, m.created_at DESC`,
		userID,
	); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if conversations == nil {
		conversations = []model.ConversationResponse{}
	}
	return c.JSON(conversations)
}

// MarkMessageAsRead handles PATCH /messages/:id/read.
//
// @Summary         Mark message as read
// @Description     Marks a single message as read. Only the receiver may mark a message as read.
// @Tags            Messages
// @Produce         json
// @Param           id path string true "Message ID"
// @Success         204
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              markMessageAsRead
// @Security        BearerAuth
// @Router          /messages/{id}/read [patch]
func MarkMessageAsRead(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid message id")
	}

	res, err := db.Exec(`
		UPDATE messages SET is_read = TRUE
		WHERE id = $1 AND receiver_id = $2`, id, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fiber.NewError(fiber.StatusNotFound, "message not found or not addressed to you")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
