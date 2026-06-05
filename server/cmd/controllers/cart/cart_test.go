package cart_test

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
	"shbs-server/cmd/controllers/cart"
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
		tcpostgres.WithDatabase("shbs_test_cart"),
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
	cart.Mount(api, testDB)

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
	return token
}

func createListing(t *testing.T, token string, title string) string {
	payload := map[string]any{
		"title":     title,
		"author":    "Author",
		"price":     "10.00",
		"condition": "good",
	}
	res, err := testApp.Test(authReq("POST", "/api/v1/listings", payload, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)
	var body map[string]any
	decodeBody(t, res, &body)
	l, _ := body["listing"].(map[string]any)
	return l["id"].(string)
}

func TestCart_Flow(t *testing.T) {
	email, pass := "cart_test@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	listingID := createListing(t, token, "Cart Book")

	// 1. Add to cart
	res, err := testApp.Test(authReq("POST", "/api/v1/cart", map[string]string{"listing_id": listingID}, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	// 2. List cart
	res, err = testApp.Test(authReq("GET", "/api/v1/cart", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var listBody map[string]any
	decodeBody(t, res, &listBody)
	items, _ := listBody["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 item in cart, got %d", len(items))
	}
	item, _ := items[0].(map[string]any)
	cartItemID := item["id"].(string)
	l, _ := item["listing"].(map[string]any)
	if l["id"] != listingID {
		t.Fatalf("expected listing id %s, got %v", listingID, l["id"])
	}

	// 3. Remove from cart
	res, err = testApp.Test(authReq("DELETE", "/api/v1/cart/"+cartItemID, nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	// 4. Verify empty cart
	res, err = testApp.Test(authReq("GET", "/api/v1/cart", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	decodeBody(t, res, &listBody)
	items, _ = listBody["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("expected 0 items in cart, got %d", len(items))
	}
}
