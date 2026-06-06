package model_test

// Tests for pkg/model verify the two most security-critical JSON serialization
// rules:
//   1. User.PasswordHash is NEVER included in JSON output (OWASP A02).
//   2. Image.S3Key is NEVER included in JSON output (internal storage key).
//
// These are not "just documentation" — a missing json:"-" tag would silently
// expose sensitive data to API clients. Catching it here keeps the CI gate
// honest even before the auth controller is written.

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"shbs-server/pkg/model"
)

func TestUser_PasswordHashNotInJSON(t *testing.T) {
	hash := "supersecretbcrypthash"
	u := model.User{
		ID:           uuid.New(),
		Name:         "Alice",
		Email:        "alice@mail.ndhu.edu.tw",
		PasswordHash: &hash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := raw["password_hash"]; ok {
		t.Error("password_hash must not appear in JSON output — OWASP A02 violation")
	}
	if _, ok := raw["PasswordHash"]; ok {
		t.Error("PasswordHash must not appear in JSON output — OWASP A02 violation")
	}
}

func TestImage_S3KeyNotInJSON(t *testing.T) {
	img := model.Image{
		ID:        uuid.New(),
		S3Key:     "uploads/books/cover-abc123.jpg",
		CdnURL:    "https://d1234.cloudfront.net/uploads/books/cover-abc123.jpg",
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(img)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if _, ok := raw["s3_key"]; ok {
		t.Error("s3_key must not appear in JSON output — internal storage key must stay server-side")
	}
	if _, ok := raw["S3Key"]; ok {
		t.Error("S3Key must not appear in JSON output — internal storage key must stay server-side")
	}
	if _, ok := raw["cdn_url"]; !ok {
		t.Error("cdn_url must appear in JSON output")
	}
}

func TestListingConditionConstants(t *testing.T) {
	valid := map[string]bool{
		model.ConditionGood:     true,
		model.ConditionModerate: true,
		model.ConditionPoor:     true,
	}
	for v := range valid {
		if v == "" {
			t.Errorf("condition constant must not be empty string")
		}
	}
}

func TestOrderStatusConstants(t *testing.T) {
	valid := map[string]bool{
		model.OrderStatusPending:   true,
		model.OrderStatusConfirmed: true,
		model.OrderStatusCompleted: true,
		model.OrderStatusCancelled: true,
	}
	for v := range valid {
		if v == "" {
			t.Errorf("order status constant must not be empty string")
		}
	}
}

func TestVerificationTypeConstants(t *testing.T) {
	types := []string{
		model.VerificationTypeEmailVerification,
		model.VerificationTypePasswordReset,
	}
	for _, v := range types {
		if v == "" {
			t.Errorf("verification type constant must not be empty string")
		}
	}
}

func TestListingStatusConstants(t *testing.T) {
	statuses := []string{
		model.ListingStatusActive,
		model.ListingStatusReserved,
		model.ListingStatusSold,
		model.ListingStatusDelisted,
	}
	for _, v := range statuses {
		if v == "" {
			t.Errorf("listing status constant must not be empty string")
		}
	}
}

func TestStringSlice_ScanPostgresArray(t *testing.T) {
	var values model.StringSlice
	if err := values.Scan([]byte(`{"https://cdn.example.com/a.jpg","https://cdn.example.com/b.jpg"}`)); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	want := model.StringSlice{"https://cdn.example.com/a.jpg", "https://cdn.example.com/b.jpg"}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("unexpected values: got %v want %v", values, want)
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	types := []string{
		model.NotifTypeNewMessage,
		model.NotifTypeOrderConfirmed,
		model.NotifTypeListingSold,
	}
	for _, v := range types {
		if v == "" {
			t.Errorf("notification type constant must not be empty string")
		}
	}
}
