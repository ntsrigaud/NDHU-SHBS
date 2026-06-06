package auth_test

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
	"shbs-server/cmd/middleware"
	"shbs-server/pkg/services"
	"shbs-server/pkg/util"
)

// ─── Suite state (shared across all tests in this package) ───────────────────

var (
	testDB  *sqlx.DB
	testApp *fiber.App
)

// TestMain starts a PostgreSQL testcontainer, runs migrations, wires the auth
// routes into a minimal Fiber app, and then runs the test suite.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("shbs_test"),
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

	// Required by JWT generation / config validation.
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
	testApp.Post("/api/v1/auth/logout", authMW, func(c *fiber.Ctx) error { return auth.LogoutUser(testDB, c) })
	testApp.Get("/api/v1/auth/verify", func(c *fiber.Ctx) error { return auth.VerifyEmail(testDB, c) })
	testApp.Post("/api/v1/auth/resend-verification", func(c *fiber.Ctx) error { return auth.ResendVerification(testDB, emailSvc, c) })

	os.Exit(m.Run())
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// applyMigrations creates the schema_migrations table and runs every .sql file
// in the migrations/ directory adjacent to this package's source tree.
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

// jsonReq builds a POST request with a JSON body.
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

// must500 asserts that res has the given HTTP status code.
func assertStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	if res.StatusCode != want {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("want status %d, got %d; body: %s", want, res.StatusCode, body)
	}
}

// decodeBody decodes the response body as JSON into dest.
func decodeBody(t *testing.T, res *http.Response, dest any) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(dest); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

// ─── Register ─────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	req := jsonReq("POST", "/api/v1/auth/register", map[string]string{
		"name":     "Test User",
		"email":    "411221367@gms.ndhu.edu.tw",
		"password": "Str0ngP@ssword!",
	})
	res, err := testApp.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	var body map[string]any
	decodeBody(t, res, &body)

	msg, ok := body["message"].(string)
	if !ok {
		t.Fatal("response missing 'message' field")
	}
	if msg != "an email was sent to your account, please verify it before logging in" {
		t.Errorf("unexpected message: %v", msg)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	payload := map[string]string{
		"name":     "Dup User",
		"email":    "411221368@gms.ndhu.edu.tw",
		"password": "Str0ngP@ssword!",
	}
	first, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/register", payload), -1)
	assertStatus(t, first, http.StatusCreated)

	second, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/register", payload), -1)
	assertStatus(t, second, http.StatusConflict)
}

func TestRegister_WeakPassword(t *testing.T) {
	req := jsonReq("POST", "/api/v1/auth/register", map[string]string{
		"name":     "Weak",
		"email":    "411221369@gms.ndhu.edu.tw",
		"password": "abc",
	})
	res, _ := testApp.Test(req, -1)
	assertStatus(t, res, http.StatusUnprocessableEntity)
}

func TestRegister_InvalidStudentEmail(t *testing.T) {
	req := jsonReq("POST", "/api/v1/auth/register", map[string]string{
		"name":     "Wrong Domain",
		"email":    "student@gmail.com",
		"password": "Str0ngP@ssword!",
	})
	res, _ := testApp.Test(req, -1)
	assertStatus(t, res, http.StatusUnprocessableEntity)

	var body map[string]any
	decodeBody(t, res, &body)

	if body["error"] != "email must be a valid NDHU student address in the format student_id@gms.ndhu.edu.tw" {
		t.Fatalf("unexpected error: %v", body["error"])
	}
}

func TestRegister_MissingName(t *testing.T) {
	req := jsonReq("POST", "/api/v1/auth/register", map[string]string{
		"name":     "",
		"email":    "411221370@gms.ndhu.edu.tw",
		"password": "Str0ngP@ssword!",
	})
	res, _ := testApp.Test(req, -1)
	assertStatus(t, res, http.StatusUnprocessableEntity)
}

// ─── Login ────────────────────────────────────────────────────────────────────

// insertVerifiedUser directly inserts a pre-verified user with a known password.
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

// insertUnverifiedUser directly inserts a user that has NOT verified their email.
func insertUnverifiedUser(t *testing.T, email, plainPassword string) {
	t.Helper()
	hash, err := util.HashPassword(plainPassword)
	if err != nil {
		t.Fatal(err)
	}
	_, err = testDB.Exec(`
		INSERT INTO users (id, name, email, password_hash, email_verified)
		VALUES ($1, $2, $3, $4, false)`,
		uuid.New(), strings.Split(email, "@")[0], email, hash,
	)
	if err != nil {
		t.Fatal("insertUnverifiedUser:", err)
	}
}

func TestLogin_Success(t *testing.T) {
	email := "login_success@test.com"
	pass := "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)

	res, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": pass,
	}), -1)
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	if body["token"] == nil {
		t.Error("expected 'token' in response")
	}
	// Check cookie was set
	cookie := res.Header.Get("Set-Cookie")
	if !strings.Contains(cookie, "jwt=") {
		t.Error("expected jwt cookie in Set-Cookie header")
	}
}

func TestLogin_UnverifiedEmail(t *testing.T) {
	email := "login_unverified@test.com"
	pass := "Str0ngP@ssword!"
	insertUnverifiedUser(t, email, pass)

	res, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": pass,
	}), -1)
	assertStatus(t, res, http.StatusForbidden)
}

func TestLogin_WrongPassword(t *testing.T) {
	email := "login_wrongpw@test.com"
	insertVerifiedUser(t, email, "Str0ngP@ssword!")

	res, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": "WrongPassword1!",
	}), -1)
	// Same 401 as "user not found" to prevent user enumeration
	assertStatus(t, res, http.StatusUnauthorized)
}

func TestLogin_UnknownEmail(t *testing.T) {
	res, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    "ghost@test.com",
		"password": "Str0ngP@ssword!",
	}), -1)
	assertStatus(t, res, http.StatusUnauthorized)
}

// ─── Verify ───────────────────────────────────────────────────────────────────

// insertPendingVerification inserts a user (email_verified=false) and a
// verification token with the given rawToken, returning the user ID.
func insertPendingVerification(t *testing.T, email, rawToken string, ttl time.Duration) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := testDB.Exec(`
		INSERT INTO users (id, name, email, password_hash, email_verified)
		VALUES ($1, $2, $3, $4, false)`,
		userID, "Pending", email, nil,
	)
	if err != nil {
		t.Fatal("insert pending user:", err)
	}
	_, err = testDB.Exec(`
		INSERT INTO verification_tokens (id, user_id, token_hash, type, expires_at)
		VALUES ($1, $2, $3, 'email_verification', $4)`,
		uuid.New(), userID, util.HashToken(rawToken), time.Now().Add(ttl),
	)
	if err != nil {
		t.Fatal("insert verification token:", err)
	}
	return userID
}

func TestVerify_Success(t *testing.T) {
	rawToken := "verify_success_token_0000000000"
	userID := insertPendingVerification(t, "verify_success@test.com", rawToken, 24*time.Hour)

	res, _ := testApp.Test(httptest.NewRequest("GET",
		"/api/v1/auth/verify?token="+rawToken, nil), -1)

	// Handler redirects with 302; browser would follow to /login?verified=true
	assertStatus(t, res, http.StatusFound)

	// Confirm DB was updated
	var verified bool
	testDB.QueryRow(`SELECT email_verified FROM users WHERE id = $1`, userID).Scan(&verified)
	if !verified {
		t.Error("user should be email_verified=true after successful verification")
	}

	// Token should be marked as used
	var usedAt *time.Time
	testDB.QueryRow(
		`SELECT used_at FROM verification_tokens WHERE token_hash = $1`,
		util.HashToken(rawToken),
	).Scan(&usedAt)
	if usedAt == nil {
		t.Error("verification token used_at should not be NULL after verify")
	}
}

func TestVerify_ExpiredToken(t *testing.T) {
	rawToken := "verify_expired_token_0000000000"
	insertPendingVerification(t, "verify_expired@test.com", rawToken, -1*time.Hour) // already expired

	res, _ := testApp.Test(httptest.NewRequest("GET",
		"/api/v1/auth/verify?token="+rawToken, nil), -1)
	assertStatus(t, res, http.StatusBadRequest)
}

func TestVerify_AlreadyUsedToken(t *testing.T) {
	rawToken := "verify_used_token_00000000000000"
	userID := insertPendingVerification(t, "verify_used@test.com", rawToken, 24*time.Hour)

	// Mark the token as already used
	testDB.Exec(`UPDATE verification_tokens SET used_at = NOW() WHERE
		token_hash = $1`, util.HashToken(rawToken))
	// And mark user verified too
	testDB.Exec(`UPDATE users SET email_verified = true WHERE id = $1`, userID)

	res, _ := testApp.Test(httptest.NewRequest("GET",
		"/api/v1/auth/verify?token="+rawToken, nil), -1)
	assertStatus(t, res, http.StatusBadRequest)
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestLogout_BlacklistsToken(t *testing.T) {
	email := "logout_test@test.com"
	pass := "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)

	// 1. Login to get a JWT
	loginRes, _ := testApp.Test(jsonReq("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": pass,
	}), -1)
	assertStatus(t, loginRes, http.StatusOK)

	var loginBody map[string]any
	decodeBody(t, loginRes, &loginBody)
	token, _ := loginBody["token"].(string)
	if token == "" {
		t.Fatal("no token in login response")
	}

	// 2. Logout using the JWT in the Authorization header
	logoutReq := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutRes, _ := testApp.Test(logoutReq, -1)
	assertStatus(t, logoutRes, http.StatusOK)

	// 3. A second logout with the same token should be rejected (token blacklisted)
	logoutReq2 := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	logoutReq2.Header.Set("Authorization", "Bearer "+token)
	logoutRes2, _ := testApp.Test(logoutReq2, -1)
	assertStatus(t, logoutRes2, http.StatusUnauthorized)
}
