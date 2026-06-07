package ai

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/services"
)

// AnalyzeCondition proxies an image condition analysis request to the AI
// microservice. The client sends an image UUID; the server resolves it to a CDN
// URL, calls the AI service, and returns the result.
//
// @Summary         Analyze book condition
// @Description     Classifies the condition of a book image using the AI microservice. Accepts the ID of an already-uploaded image.
// @Tags            AI
// @Accept          json
// @Param           body body model.SwaggerAnalyzeConditionRequest true "Image ID to analyze"
// @Produce         json
// @Success         200 {object} model.SwaggerAnalyzeConditionResponse
// @Failure         400 {object} model.SwaggerErrorResponse
// @Failure         401 {object} model.SwaggerErrorResponse
// @Failure         404 {object} model.SwaggerErrorResponse
// @Failure         502 {object} model.SwaggerErrorResponse
// @ID              analyzeCondition
// @Security        BearerAuth
// @Router          /ai/condition [post]
func AnalyzeCondition(db *sqlx.DB, c *fiber.Ctx) error {
	var req model.SwaggerAnalyzeConditionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	imageID, err := uuid.Parse(req.ImageID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image_id must be a valid UUID"})
	}

	var img model.Image
	if err := db.Get(&img, `SELECT id, s3_key, cdn_url, created_at FROM images WHERE id = $1`, imageID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
	}

	aiSvc := services.NewAIService("")
	res, err := aiSvc.AnalyzeCondition([]string{img.CdnURL})
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": fmt.Sprintf("AI service error: %v", err)})
	}

	return c.JSON(model.SwaggerAnalyzeConditionResponse{
		Condition:  res.Condition,
		Confidence: res.Confidence,
	})
}
