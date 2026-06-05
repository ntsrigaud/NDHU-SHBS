package util_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"shbs-server/pkg/util"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const testSecret = "test-secret-that-is-at-least-32-chars-long!"

func setJWTEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("JWT_EXPIRY_HOURS", "1")
}

func fiberApp() *fiber.App {
	return fiber.New(fiber.Config{DisableStartupMessage: true})
}

// ── GenerateJWT / ParseJWT ────────────────────────────────────────────────────

func TestGenerateJWT_ValidToken(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, expiresAt, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateJWT returned empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("ExpiresAt is in the past")
	}
}

func TestParseJWT_ValidClaims(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, true)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}
	claims, err := util.ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %s, want %s", claims.UserID, userID)
	}
	if !claims.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestParseJWT_InvalidToken(t *testing.T) {
	setJWTEnv(t)
	_, err := util.ParseJWT("not.a.valid.token")
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestParseJWT_WrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("JWT_EXPIRY_HOURS", "1")

	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}
	os.Setenv("JWT_SECRET", "completely-different-secret-value-xx!!")
	_, err = util.ParseJWT(token)
	if err == nil {
		t.Fatal("Expected error for wrong secret, got nil")
	}
}

func TestGenerateJWT_DefaultExpiry(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	os.Unsetenv("JWT_EXPIRY_HOURS")

	userID := uuid.New()
	_, expiresAt, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT error: %v", err)
	}
	expected := time.Now().Add(23 * time.Hour)
	if expiresAt.Before(expected) {
		t.Errorf("ExpiresAt %v is earlier than expected minimum %v", expiresAt, expected)
	}
}

// ── ExtractClaims / ExtractUserIDFromJwtToken / extractRawToken ──────────────

func buildExtractApp(t *testing.T) func(req *http.Request) *http.Response {
	t.Helper()
	app := fiberApp()

	app.Get("/extract-claims", func(c *fiber.Ctx) error {
		claims, err := util.ExtractClaims(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"user_id": claims.UserID.String(), "is_admin": claims.IsAdmin})
	})

	app.Get("/extract-userid", func(c *fiber.Ctx) error {
		userID, err := util.ExtractUserIDFromJwtToken(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"user_id": userID.String()})
	})

	return func(req *http.Request) *http.Response {
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("app.Test error: %v", err)
		}
		return resp
	}
}

func TestExtractClaims_FromCookie(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-claims", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	resp := do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExtractClaims_FromBearer(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, true)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-claims", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExtractClaims_MissingToken(t *testing.T) {
	setJWTEnv(t)
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-claims", nil)
	resp := do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestExtractClaims_InvalidToken(t *testing.T) {
	setJWTEnv(t)
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-claims", nil)
	req.Header.Set("Authorization", "Bearer not.valid.token")
	resp := do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestExtractUserIDFromJwtToken_Cookie(t *testing.T) {
	setJWTEnv(t)
	userID := uuid.New()
	token, _, err := util.GenerateJWT(userID, false)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-userid", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: token})
	resp := do(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExtractUserIDFromJwtToken_Missing(t *testing.T) {
	setJWTEnv(t)
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-userid", nil)
	resp := do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestExtractClaims_MalformedBearer(t *testing.T) {
	setJWTEnv(t)
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-claims", nil)
	req.Header.Set("Authorization", "Bearer")
	resp := do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for malformed bearer, got %d", resp.StatusCode)
	}
}

// TestExtractUserIDFromJwtToken_InvalidToken covers the ParseJWT-error branch
// ("return uuid.Nil, err") that is not reached by the missing-token test.
func TestExtractUserIDFromJwtToken_InvalidToken(t *testing.T) {
	setJWTEnv(t)
	do := buildExtractApp(t)
	req := httptest.NewRequest(http.MethodGet, "/extract-userid", nil)
	req.Header.Set("Authorization", "Bearer bad.token.value")
	resp := do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
	}
}
