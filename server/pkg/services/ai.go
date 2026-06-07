package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// MetadataResult represents the response from the AI metadata service.
type MetadataResult struct {
	Title      *string `json:"title"`
	Author     *string `json:"author"`
	ISBN       *string `json:"isbn"`
	Confidence float64 `json:"confidence"`
}

// ConditionResult represents the response from the AI condition service.
type ConditionResult struct {
	Condition  string  `json:"condition"`
	Score      float64 `json:"score"`
	Confidence float64 `json:"confidence"`
}

// AIService handles communication with the AI microservice.
type AIService struct {
	BaseURL string
	Client  *http.Client
}

// NewAIService creates a new AIService with a configured timeout.
func NewAIService(baseURL string) *AIService {
	if baseURL == "" {
		baseURL = os.Getenv("AI_SERVICE_URL")
	}
	return &AIService{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

// AnalyzeMetadata extracts book information from image URLs.
func (s *AIService) AnalyzeMetadata(imageURLs []string) (*MetadataResult, error) {
	if s.BaseURL == "" {
		return nil, fmt.Errorf("AI service URL not configured")
	}

	payload, _ := json.Marshal(map[string]any{
		"image_urls": imageURLs,
	})

	resp, err := s.Client.Post(
		fmt.Sprintf("%s/analyze/metadata", s.BaseURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status: %d", resp.StatusCode)
	}

	var result MetadataResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AnalyzeCondition classifies book condition from image URLs.
func (s *AIService) AnalyzeCondition(imageURLs []string) (*ConditionResult, error) {
	if s.BaseURL == "" {
		return nil, fmt.Errorf("AI service URL not configured")
	}

	payload, _ := json.Marshal(map[string]any{
		"image_urls": imageURLs,
	})

	resp, err := s.Client.Post(
		fmt.Sprintf("%s/analyze/condition", s.BaseURL),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned status: %d", resp.StatusCode)
	}

	var result ConditionResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
