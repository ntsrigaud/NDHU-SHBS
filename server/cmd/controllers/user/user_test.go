package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"shbs-server/cmd/controllers/auth"
	"shbs-server/cmd/controllers/user"
	"shbs-server/cmd/middleware"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"
)

var (
	testDB  *sqlx.DB
	testApp *fiber.App
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("shbs_test_user"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("get connection string: " + err.Error())
	}

	testDB, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		panic("connect to test db: " + err.Error())
	}
	defer testDB.Close()

	applyMigrations(testDB)

	os.Setenv("JWT_SECRET", "test-secret-that-is-definitely-32chars!")
	os.Setenv("JWT_EXPIRY_HOURS", "24")
	os.Setenv("FRONTEND_BASE_URL", "http://localhost:3000")
	os.Setenv("API_BASE_URL", "http://localhost:8080")

	testApp = fiber.New()
	testApp.Use(middleware.ErrorHandler())
	emailSvc := &services.EmailService{}
	authMW := middleware.Auth(testDB)
	testApp.Post("/api/v1/auth/register", func(c *fiber.Ctx) error { return auth.RegisterUser(testDB, emailSvc, c) })
	testApp.Post("/api/v1/auth/login", func(c *fiber.Ctx) error { return auth.LoginUser(testDB, c) })
	testApp.Post("/api/v1/auth/logout", func(c *fiber.Ctx) error { return auth.LogoutUser(testDB, c) })
	testApp.Get("/api/v1/auth/verify", func(c *fiber.Ctx) error { return auth.VerifyEmail(testDB, c) })
	testApp.Post("/api/v1/auth/resend-verification", func(c *fiber.Ctx) error { return auth.ResendVerification(testDB, emailSvc, c) })
	testApp.Get("/api/v1/users/me", authMW, func(c *fiber.Ctx) error { return user.GetMe(testDB, c) })
	testApp.Put("/api/v1/users/me", authMW, func(c *fiber.Ctx) error { return user.UpdateMe(testDB, c) })

	os.Exit(m.Run())
}

func applyMigrations(db *sqlx.DB) {
	db.MustExec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename   TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)

	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "../../../migrations")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		panic("read migrations dir: " + err.Error())
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		sql, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			panic("read " + name + ": " + err.Error())
		}
		tx := db.MustBegin()
		if _, err := tx.Exec(string(sql)); err != nil {
			_ = tx.Rollback()
			panic("apply " + name + ": " + err.Error())
		}
		_, _ = tx.Exec(`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, name)
		_ = tx.Commit()
	}
}

func jsonReq(method, path string, body any) *http.Request {
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	if res.StatusCode != want {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("want status %d, got %d; body: %s", want, res.StatusCode, body)
	}
}

func decodeBody(t *testing.T, res *http.Response, dest any) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func insertVerifiedUser(t *testing.T, email, plainPassword string) {
	t.Helper()
	hash, err := util.HashPassword(plainPassword)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testDB.Exec(`
		INSERT INTO users (id, name, email, password_hash, email_verified)
		VALUES ($1, $2, $3, $4, true)`,
		uuid.New(), strings.Split(email, "@")[0], email, hash,
	)
	if err != nil {
		t.Fatal("insertVerifiedUser:", err)
	}
}

func loginAndGetToken(t *testing.T, email, password string) string {
	t.Helper()
	res, err := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	token, _ := body["token"].(string)
	if token == "" {
		t.Fatal("no token in login response")
	}
	return token
}

func TestGetMe_Unauthorized(t *testing.T) {
	res, err := testApp.Test(httptest.NewRequest("GET", "/api/v1/users/me", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnauthorized)
}

func TestGetMe_Success(t *testing.T) {
	email := "me_success@test.com"
	pass := "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	req := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	if body["email"] != email {
		t.Fatalf("unexpected email: %v", body["email"])
	}
}

func TestUpdateMe_NameSuccess(t *testing.T) {
	email := "update_success@test.com"
	pass := "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	req := jsonReq("PUT", "/api/v1/users/me", map[string]string{"name": "Updated Name"})
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	if body["name"] != "Updated Name" {
		t.Fatalf("name not updated, got: %v", body["name"])
	}
}

func TestUpdateMe_InvalidAvatarUUID(t *testing.T) {
	email := "update_avatar_invalid@test.com"
	pass := "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	req := jsonReq("PUT", "/api/v1/users/me", map[string]string{"avatar_image_id": "not-a-uuid"})
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnprocessableEntity)
}
