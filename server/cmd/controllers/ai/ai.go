package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
)

// aiConditionRequest mirrors the AI service's expected payload.
type aiConditionRequest struct {
	ImageURLs []string `json:"image_urls"`
}

// aiConditionResponse mirrors the AI service's response.
type aiConditionResponse struct {
	Condition  string  `json:"condition"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"` // some versions use confidence
}

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

	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "AI service not configured"})
	}

	payload, _ := json.Marshal(aiConditionRequest{ImageURLs: []string{img.CdnURL}})

	resp, err := http.Post(
		fmt.Sprintf("%s/analyze/condition", aiServiceURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "AI service unavailable"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "AI service returned an error"})
	}

	var aiResp aiConditionResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "invalid response from AI service"})
	}

	// Normalise: prefer "confidence" field, fall back to "score".
	confidence := aiResp.Confidence
	if confidence == 0 {
		confidence = aiResp.Score
	}

	return c.JSON(model.SwaggerAnalyzeConditionResponse{
		Condition:  aiResp.Condition,
		Confidence: confidence,
	})
}
