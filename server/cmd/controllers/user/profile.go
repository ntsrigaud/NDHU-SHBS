package user

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// GetMe returns the authenticated user's profile.
//
// @Summary         Get current user
// @Description     Returns the authenticated user's profile information
// @Tags            Users
// @Produce         json
// @Success         200 {object} model.User
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @ID              getMe
// @Security        BearerAuth
// @Router          /users/me [get]
func GetMe(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var user model.User
	if err := db.Get(&user, `SELECT * FROM users WHERE id = $1`, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(user)
}

// UpdateMe updates editable fields on the authenticated user's profile.
//
// @Summary         Update current user
// @Description     Updates the authenticated user's name and/or avatar image
// @Tags            Users
// @Accept          json
// @Param           body body model.SwaggerUpdateUserRequest true "Fields to update"
// @Produce         json
// @Success         200 {object} model.User
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              updateMe
// @Security        BearerAuth
// @Router          /users/me [put]
func UpdateMe(db *sqlx.DB, c *fiber.Ctx) error {
	userID, err := userIDFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req model.SwaggerUpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		req.Name = &n
	}
	if req.AvatarImageID != nil {
		a := strings.TrimSpace(*req.AvatarImageID)
		req.AvatarImageID = &a
	}

	if req.Name == nil && req.AvatarImageID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no fields to update"})
	}
	if req.Name != nil && *req.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is required"})
	}

	var avatarID *uuid.UUID
	setAvatar := req.AvatarImageID != nil
	if req.AvatarImageID != nil && *req.AvatarImageID != "" {
		parsed, err := uuid.Parse(*req.AvatarImageID)
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "avatar_image_id must be a valid UUID"})
		}
		var exists bool
		if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM images WHERE id = $1)`, parsed).Scan(&exists); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database error"})
		}
		if !exists {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "avatar image not found"})
		}
		avatarID = &parsed
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
	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d RETURNING *`, strings.Join(setClauses, ", "), i)

	var user model.User
	if err := db.QueryRowx(query, args...).StructScan(&user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not update user"})
	}

	return c.JSON(user)
}

func userIDFromContext(c *fiber.Ctx) (uuid.UUID, error) {
	raw, ok := c.Locals("userID").(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return uuid.Nil, fmt.Errorf("missing userID in context")
	}
	return uuid.Parse(raw)
}
