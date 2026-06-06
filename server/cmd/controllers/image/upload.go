package image

import (
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/services"
)

const (
	maxImageSize = 5 * 1024 * 1024 // 5 MB
)

var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// UploadImage handles POST /images/upload (auth required).
//
// @Summary         Upload image
// @Description     Accepts a multipart file upload, stores it in S3, registers it in the database, and returns the image ID and CDN URL.
// @Tags            Images
// @Accept          mpfd
// @Param           file formData file true "Image file (JPEG, PNG, WebP; max 5 MB)"
// @Produce         json
// @Success         201 {object} model.SwaggerUploadImageResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         413 {object} model.SwaggerErrorResponse
// @Failure         422 {object} model.SwaggerErrorResponse
// @Failure         500 {object} model.SwaggerErrorResponse
// @ID              uploadImage
// @Security        BearerAuth
// @Router          /images/upload [post]
func UploadImage(db *sqlx.DB, s3Svc *services.S3Service, c *fiber.Ctx) error {
	if s3Svc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "image upload service is not configured"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}

	if fileHeader.Size > maxImageSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file must be ≤ 5 MB"})
	}

	lastDot := strings.LastIndex(fileHeader.Filename, ".")
	if lastDot < 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "file must have an extension"})
	}
	ext := strings.ToLower(fileHeader.Filename[lastDot:])
	if !allowedExts[ext] {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "only JPEG, PNG, and WebP images are accepted"})
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not read uploaded file"})
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not read uploaded file"})
	}

	s3Key, cdnURL, err := s3Svc.UploadImage(c.Context(), data, fileHeader.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not upload image"})
	}

	var img model.Image
	err = db.QueryRowx(
		`INSERT INTO images (id, s3_key, cdn_url) VALUES ($1, $2, $3) RETURNING id, s3_key, cdn_url, created_at`,
		uuid.New(), s3Key, cdnURL,
	).StructScan(&img)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not register image"})
	}

	return c.Status(fiber.StatusCreated).JSON(model.SwaggerUploadImageResponse{
		Image: model.ImageResponse{
			ID:     img.ID,
			CdnURL: img.CdnURL,
		},
	})
}
