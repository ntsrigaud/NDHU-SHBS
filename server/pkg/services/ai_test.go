package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewAIService(t *testing.T) {
	t.Run("with explicit URL", func(t *testing.T) {
		svc := NewAIService("http://explicit.ai")
		if svc.BaseURL != "http://explicit.ai" {
			t.Errorf("expected http://explicit.ai, got %s", svc.BaseURL)
		}
	})

	t.Run("from environment", func(t *testing.T) {
		t.Setenv("AI_SERVICE_URL", "http://env.ai")
		svc := NewAIService("")
		if svc.BaseURL != "http://env.ai" {
			t.Errorf("expected http://env.ai, got %s", svc.BaseURL)
		}
	})
}

func TestAIService_AnalyzeMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		title := "Test Book"
		author := "Test Author"
		isbn := "1234567890123"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/analyze/metadata" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(MetadataResult{
				Title:      &title,
				Author:     &author,
				ISBN:       &isbn,
				Confidence: 0.95,
			})
		}))
		defer server.Close()

		svc := NewAIService(server.URL)
		res, err := svc.AnalyzeMetadata([]string{"http://image.url"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *res.Title != title || *res.Author != author || *res.ISBN != isbn || res.Confidence != 0.95 {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		svc := NewAIService(server.URL)
		_, err := svc.AnalyzeMetadata([]string{"http://image.url"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		svc := NewAIService(server.URL)
		_, err := svc.AnalyzeMetadata([]string{"http://image.url"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAIService_AnalyzeCondition(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/analyze/condition" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			json.NewEncoder(w).Encode(ConditionResult{
				Condition:  "good",
				Score:      0.9,
				Confidence: 0.85,
			})
		}))
		defer server.Close()

		svc := NewAIService(server.URL)
		res, err := svc.AnalyzeCondition([]string{"http://image.url"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Condition != "good" || res.Score != 0.9 || res.Confidence != 0.85 {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("no base url", func(t *testing.T) {
		svc := NewAIService("")
		_, err := svc.AnalyzeCondition([]string{"http://image.url"})
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "AI service URL not configured" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("network error", func(t *testing.T) {
		svc := NewAIService("http://invalid-url-that-does-not-exist.local")
		_, err := svc.AnalyzeCondition([]string{"http://image.url"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
