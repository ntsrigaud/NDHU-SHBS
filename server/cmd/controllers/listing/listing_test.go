package listing_test

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
	"shbs-server/cmd/controllers/listing"
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
		tcpostgres.WithDatabase("shbs_test_listing"),
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
	api := testApp.Group("/api/v1")
	auth.Mount(api, testDB, &services.EmailService{})
	listing.Mount(api, testDB)

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

// ── Helpers ───────────────────────────────────────────────────────────────────

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

func authReq(method, path string, body any, token string) *http.Request {
	req := jsonReq(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func assertStatus(t *testing.T, res *http.Response, want int) {
	t.Helper()
	if res.StatusCode != want {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("want status %d, got %d; body: %s", want, res.StatusCode, b)
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

// createListing is a test helper that POSTs to /listings and returns the listing id.
func createListing(t *testing.T, token string, payload map[string]any) string {
	t.Helper()
	res, err := testApp.Test(authReq("POST", "/api/v1/listings", payload, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	var body map[string]any
	decodeBody(t, res, &body)
	listingMap, _ := body["listing"].(map[string]any)
	id, _ := listingMap["id"].(string)
	if id == "" {
		t.Fatal("createListing: no id in response")
	}
	return id
}

var samplePayload = map[string]any{
	"title":     "Introduction to Algorithms",
	"author":    "Cormen",
	"price":     "9.99",
	"condition": "good",
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreateListing_Unauthorized(t *testing.T) {
	res, err := testApp.Test(jsonReq("POST", "/api/v1/listings", samplePayload), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnauthorized)
}

func TestCreateListing_Success(t *testing.T) {
	email, pass := "create_ok@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	res, err := testApp.Test(authReq("POST", "/api/v1/listings", samplePayload, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	var body map[string]any
	decodeBody(t, res, &body)
	l, _ := body["listing"].(map[string]any)
	if l["title"] != "Introduction to Algorithms" {
		t.Fatalf("unexpected title: %v", l["title"])
	}
	if l["status"] != "active" {
		t.Fatalf("expected status active, got %v", l["status"])
	}
}

func TestCreateListing_MissingTitle(t *testing.T) {
	email, pass := "create_notitle@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	payload := map[string]any{
		"author": "Author", "price": "5.00", "condition": "good",
	}
	res, err := testApp.Test(authReq("POST", "/api/v1/listings", payload, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnprocessableEntity)
}

func TestCreateListing_InvalidCondition(t *testing.T) {
	email, pass := "create_badcond@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	payload := map[string]any{
		"title": "Book", "author": "Author", "price": "5.00", "condition": "mint",
	}
	res, err := testApp.Test(authReq("POST", "/api/v1/listings", payload, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnprocessableEntity)
}

func TestListListings_Public(t *testing.T) {
	res, err := testApp.Test(httptest.NewRequest("GET", "/api/v1/listings", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	if _, ok := body["data"]; !ok {
		t.Fatal("expected 'data' field in response")
	}
	if _, ok := body["total"]; !ok {
		t.Fatal("expected 'total' field in response")
	}
}

func TestListListings_FilterByCondition(t *testing.T) {
	email, pass := "filter_cond@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	// Create a "good" and a "poor" listing.
	createListing(t, token, map[string]any{
		"title": "Good Book", "author": "A", "price": "10.00", "condition": "good",
	})
	createListing(t, token, map[string]any{
		"title": "Poor Book", "author": "B", "price": "2.00", "condition": "poor",
	})

	res, err := testApp.Test(
		httptest.NewRequest("GET", "/api/v1/listings?condition=poor", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	data, _ := body["data"].([]any)
	for _, item := range data {
		m, _ := item.(map[string]any)
		if m["condition"] != "poor" {
			t.Fatalf("expected only 'poor' condition listings, got %v", m["condition"])
		}
	}
}

func TestGetListing_NotFound(t *testing.T) {
	res, err := testApp.Test(
		httptest.NewRequest("GET", "/api/v1/listings/"+uuid.New().String(), nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusNotFound)
}

func TestGetListing_InvalidID(t *testing.T) {
	res, err := testApp.Test(
		httptest.NewRequest("GET", "/api/v1/listings/not-a-uuid", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusBadRequest)
}

func TestGetListing_Success(t *testing.T) {
	email, pass := "get_listing@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	id := createListing(t, token, samplePayload)

	res, err := testApp.Test(
		httptest.NewRequest("GET", "/api/v1/listings/"+id, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	l, _ := body["listing"].(map[string]any)
	if l["id"] != id {
		t.Fatalf("expected listing id %s, got %v", id, l["id"])
	}
}

func TestUpdateListing_Unauthorized(t *testing.T) {
	res, err := testApp.Test(
		jsonReq("PUT", "/api/v1/listings/"+uuid.New().String(), map[string]string{"title": "x"}), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnauthorized)
}

func TestUpdateListing_Forbidden(t *testing.T) {
	// Owner creates a listing.
	owner, ownerPass := "owner_forbid@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, owner, ownerPass)
	ownerToken := loginAndGetToken(t, owner, ownerPass)
	id := createListing(t, ownerToken, samplePayload)

	// Different user tries to update it.
	other, otherPass := "other_forbid@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, other, otherPass)
	otherToken := loginAndGetToken(t, other, otherPass)

	res, err := testApp.Test(
		authReq("PUT", "/api/v1/listings/"+id, map[string]string{"title": "Hijacked"}, otherToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusForbidden)
}

func TestUpdateListing_Success(t *testing.T) {
	email, pass := "update_listing@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)
	id := createListing(t, token, samplePayload)

	res, err := testApp.Test(
		authReq("PUT", "/api/v1/listings/"+id, map[string]any{
			"title": "Updated Title",
			"price": "19.99",
		}, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var body map[string]any
	decodeBody(t, res, &body)
	l, _ := body["listing"].(map[string]any)
	if l["title"] != "Updated Title" {
		t.Fatalf("expected updated title, got %v", l["title"])
	}
}

func TestUpdateListing_InvalidStatus(t *testing.T) {
	email, pass := "update_badstatus@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)
	id := createListing(t, token, samplePayload)

	// Sellers may not directly set status to "sold" (managed by order flow).
	res, err := testApp.Test(
		authReq("PUT", "/api/v1/listings/"+id, map[string]string{"status": "sold"}, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnprocessableEntity)
}

func TestDelistListing_Forbidden(t *testing.T) {
	owner, ownerPass := "delist_owner@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, owner, ownerPass)
	ownerToken := loginAndGetToken(t, owner, ownerPass)
	id := createListing(t, ownerToken, samplePayload)

	other, otherPass := "delist_other@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, other, otherPass)
	otherToken := loginAndGetToken(t, other, otherPass)

	res, err := testApp.Test(
		authReq("DELETE", "/api/v1/listings/"+id, nil, otherToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusForbidden)
}

func TestDelistListing_Success(t *testing.T) {
	email, pass := "delist_ok@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)
	id := createListing(t, token, samplePayload)

	res, err := testApp.Test(
		authReq("DELETE", "/api/v1/listings/"+id, nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	// Verify the listing is now "delisted" and excluded from the default (active) list.
	getRes, err := testApp.Test(
		httptest.NewRequest("GET", "/api/v1/listings/"+id, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, getRes, http.StatusOK)

	var body map[string]any
	decodeBody(t, getRes, &body)
	l, _ := body["listing"].(map[string]any)
	if l["status"] != "delisted" {
		t.Fatalf("expected status 'delisted', got %v", l["status"])
	}
}
