package image

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// RegisterImage handles POST /images (auth required).
//
// @Summary         Register image
// @Description     Registers an image metadata record in the database after upload to S3
// @Tags            Images
// @Accept          json
// @Param           body body model.SwaggerRegisterImageRequest true "Image metadata"
// @Produce         json
// @Success         201 {object} model.ImageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              registerImage
// @Security        BearerAuth
// @Router          /images [post]
func RegisterImage(db *sqlx.DB, c *fiber.Ctx) error {
	var req model.SwaggerRegisterImageRequest
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

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"image": model.ImageResponse{
		ID:     img.ID,
		CdnURL: img.CdnURL,
	}})
}

// GetImage handles GET /images/:id.
//
// @Summary         Get image
// @Description     Returns image metadata by ID
// @Tags            Images
// @Produce         json
// @Param           id path string true "Image ID"
// @Success         200 {object} model.ImageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @ID              getImage
// @Router          /images/{id} [get]
func GetImage(db *sqlx.DB, c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid image id")
	}

	var img model.Image
	if err := db.Get(&img, `SELECT * FROM images WHERE id = $1`, id); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "image not found")
	}

	return c.JSON(fiber.Map{"image": model.ImageResponse{
		ID:     img.ID,
		CdnURL: img.CdnURL,
	}})
}
