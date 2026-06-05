package image

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// HandleRegisterImage handles POST /images (auth required).
// This registers an image metadata record in the database.
func HandleRegisterImage(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RegisterImageRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
		}

		if req.S3Key == "" || req.CdnURL == "" {
			return fiber.NewError(fiber.StatusUnprocessableEntity, "s3_key and cdn_url are required")
		}

		var img model.Image
		err := db.QueryRowx(`
			INSERT INTO images (id, s3_key, cdn_url)
			VALUES ($1, $2, $3)
			RETURNING id, s3_key, cdn_url, created_at`,
			uuid.New(), req.S3Key, req.CdnURL,
		).StructScan(&img)

		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not register image")
		}

		return c.Status(fiber.StatusCreated).JSON(ImageResponse{
			ID:     img.ID,
			CdnURL: img.CdnURL,
		})
	}
}

// HandleGetImage handles GET /images/:id.
func HandleGetImage(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid image id")
		}

		var img model.Image
		if err := db.Get(&img, `SELECT * FROM images WHERE id = $1`, id); err != nil {
			return fiber.NewError(fiber.StatusNotFound, "image not found")
		}

		return c.JSON(ImageResponse{
			ID:     img.ID,
			CdnURL: img.CdnURL,
		})
	}
}
