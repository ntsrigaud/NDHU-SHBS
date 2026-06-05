package notification

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HandleListNotifications handles GET /notifications.
func HandleListNotifications(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var notifications []NotificationResponse
		err = db.Select(&notifications, `
			SELECT id, type, payload, is_read, created_at
			FROM notifications
			WHERE user_id = $1
			ORDER BY created_at DESC`, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		return c.JSON(fiber.Map{"notifications": notifications})
	}
}

// HandleMarkAsRead handles PATCH /notifications/:id.
func HandleMarkAsRead(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid notification id")
		}

		res, err := db.Exec(`
			UPDATE notifications 
			SET is_read = TRUE 
			WHERE id = $1 AND user_id = $2`, id, userID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "database error")
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fiber.NewError(fiber.StatusNotFound, "notification not found")
		}

		return c.JSON(fiber.Map{"message": "notification marked as read"})
	}
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
