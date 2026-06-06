package notification

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// GetNotifications handles GET /notifications.
//
// @Summary         List notifications
// @Description     Returns all notifications for the authenticated user, newest first.
// @Tags            Notifications
// @Produce         json
// @Success         200 {array} model.NotificationResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getNotifications
// @Security        BearerAuth
// @Router          /notifications [get]
func GetNotifications(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var notifications []model.NotificationResponse
	err = db.Select(&notifications, `
		SELECT id, type, payload, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if notifications == nil {
		notifications = []model.NotificationResponse{}
	}
	return c.JSON(notifications)
}

// GetUnreadNotificationCount handles GET /notifications/unread-count.
//
// @Summary         Unread notification count
// @Description     Returns the number of unread notifications for the authenticated user.
// @Tags            Notifications
// @Produce         json
// @Success         200 {object} map[string]int
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              getUnreadNotificationCount
// @Security        BearerAuth
// @Router          /notifications/unread-count [get]
func GetUnreadNotificationCount(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	var count int
	if err := db.Get(&count, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id = $1 AND is_read = FALSE`, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.JSON(fiber.Map{"count": count})
}

// MarkAllNotificationsAsRead handles PATCH /notifications/read-all.
//
// @Summary         Mark all notifications as read
// @Description     Marks every unread notification for the authenticated user as read.
// @Tags            Notifications
// @Produce         json
// @Success         204
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              markAllNotificationsAsRead
// @Security        BearerAuth
// @Router          /notifications/read-all [patch]
func MarkAllNotificationsAsRead(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if _, err := db.Exec(`
		UPDATE notifications SET is_read = TRUE
		WHERE user_id = $1 AND is_read = FALSE`, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// MarkNotificationAsRead handles PATCH /notifications/:id/read.
//
// @Summary         Mark notification as read
// @Description     Marks a single notification as read. Returns 404 if the notification does not belong to the current user.
// @Tags            Notifications
// @Produce         json
// @Param           id path string true "Notification ID"
// @Success         204
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              markNotificationAsRead
// @Security        BearerAuth
// @Router          /notifications/{id}/read [patch]
func MarkNotificationAsRead(db *sqlx.DB, c *fiber.Ctx) error {
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

	return c.SendStatus(fiber.StatusNoContent)
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
