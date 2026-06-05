package user

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// HandleGetMe returns the authenticated user's profile.
func HandleGetMe(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var user model.User
		if err := db.Get(&user, `SELECT * FROM users WHERE id = $1`, userID); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "user not found")
		}

		return c.JSON(fiber.Map{"user": user})
	}
}

// HandleUpdateMe updates editable fields on the authenticated user's profile.
func HandleUpdateMe(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, err := userIDFromContext(c)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}

		var req UpdateMeRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}
		normalizeUpdateMeRequest(&req)

		if req.Name == nil && req.AvatarImageID == nil {
			return fiber.NewError(fiber.StatusBadRequest, "no fields to update")
		}

		if req.Name != nil && *req.Name == "" {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "name is required")
		}

		var avatarID *uuid.UUID
		setAvatar := req.AvatarImageID != nil
		if req.AvatarImageID != nil {
			if *req.AvatarImageID != "" {
				parsed, err := uuid.Parse(*req.AvatarImageID)
				if err != nil {
					return fiber.NewError(fiber.StatusUnprocessableEntity, "avatar_image_id must be a valid UUID")
				}

				var exists bool
				if err := db.QueryRow(
					`SELECT EXISTS(SELECT 1 FROM images WHERE id = $1)`,
					parsed,
				).Scan(&exists); err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "database error")
				}
				if !exists {
					return fiber.NewError(fiber.StatusUnprocessableEntity, "avatar image not found")
				}
				avatarID = &parsed
			}
		}

		setClauses := []string{"updated_at = NOW()"}
		args := make([]any, 0, 4)
		i := 1

		if req.Name != nil {
			setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
			args = append(args, *req.Name)
			i++
		}
		if setAvatar {
			setClauses = append(setClauses, fmt.Sprintf("avatar_image_id = $%d", i))
			args = append(args, avatarID)
			i++
		}

		args = append(args, userID)
		query := fmt.Sprintf(
			`UPDATE users SET %s WHERE id = $%d RETURNING *`,
			strings.Join(setClauses, ", "),
			i,
		)

		var user model.User
		if err := db.QueryRowx(query, args...).StructScan(&user); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not update user")
		}

		return c.JSON(fiber.Map{"user": user})
	}
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
