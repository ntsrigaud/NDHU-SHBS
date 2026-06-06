package order_test

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
	"shbs-server/cmd/controllers/notification"
	"shbs-server/cmd/controllers/order"
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
		tcpostgres.WithDatabase("shbs_test_order"),
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
	testApp.Get("/api/v1/listings", func(c *fiber.Ctx) error { return listing.GetListings(testDB, c) })
	testApp.Get("/api/v1/listings/me", authMW, func(c *fiber.Ctx) error { return listing.GetMyListings(testDB, c) })
	testApp.Get("/api/v1/listings/:id", func(c *fiber.Ctx) error { return listing.GetListing(testDB, c) })
	testApp.Post("/api/v1/listings", authMW, func(c *fiber.Ctx) error { return listing.CreateListing(testDB, c) })
	testApp.Patch("/api/v1/listings/:id", authMW, func(c *fiber.Ctx) error { return listing.UpdateListing(testDB, c) })
	testApp.Delete("/api/v1/listings/:id", authMW, func(c *fiber.Ctx) error { return listing.DeleteListing(testDB, c) })
	testApp.Get("/api/v1/cart", authMW, func(c *fiber.Ctx) error { return cart.GetCart(testDB, c) })
	testApp.Post("/api/v1/cart", authMW, func(c *fiber.Ctx) error { return cart.AddToCart(testDB, c) })
	testApp.Delete("/api/v1/cart/:id", authMW, func(c *fiber.Ctx) error { return cart.RemoveFromCart(testDB, c) })
	testApp.Post("/api/v1/orders", authMW, func(c *fiber.Ctx) error { return order.Checkout(testDB, c) })
	testApp.Get("/api/v1/orders", authMW, func(c *fiber.Ctx) error { return order.GetOrders(testDB, c) })
	testApp.Get("/api/v1/notifications", authMW, func(c *fiber.Ctx) error { return notification.GetNotifications(testDB, c) })
	testApp.Get("/api/v1/notifications/unread-count", authMW, func(c *fiber.Ctx) error { return notification.GetUnreadNotificationCount(testDB, c) })
	testApp.Patch("/api/v1/notifications/read-all", authMW, func(c *fiber.Ctx) error { return notification.MarkAllNotificationsAsRead(testDB, c) })
	testApp.Patch("/api/v1/notifications/:id/read", authMW, func(c *fiber.Ctx) error { return notification.MarkNotificationAsRead(testDB, c) })

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

func addToCart(t *testing.T, token string, listingID string) {
	res, err := testApp.Test(authReq("POST", "/api/v1/cart", map[string]string{"listing_id": listingID}, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)
}

func TestOrder_CheckoutFlow(t *testing.T) {
	email, pass := "order_test@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	l1 := createListing(t, token, "Book 1")
	l2 := createListing(t, token, "Book 2")

	addToCart(t, token, l1)
	addToCart(t, token, l2)

	// 1. Checkout
	res, err := testApp.Test(authReq("POST", "/api/v1/orders", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	var checkoutBody map[string]any
	decodeBody(t, res, &checkoutBody)
	if checkoutBody["message"] != "order placed successfully" {
		t.Fatalf("unexpected message: %v", checkoutBody["message"])
	}

	// 2. Verify Order History
	res, err = testApp.Test(authReq("GET", "/api/v1/orders", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var historyBody map[string]any
	decodeBody(t, res, &historyBody)
	orders, _ := historyBody["orders"].([]any)
	if len(orders) != 1 {
		t.Fatalf("expected 1 order, got %d", len(orders))
	}

	o, _ := orders[0].(map[string]any)
	items, _ := o["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 items in order, got %d", len(items))
	}

	// 3. Verify Listing Status is 'sold'
	for _, id := range []string{l1, l2} {
		res, err = testApp.Test(httptest.NewRequest("GET", "/api/v1/listings/"+id, nil), -1)
		if err != nil {
			t.Fatal(err)
		}
		assertStatus(t, res, http.StatusOK)
		var lBody map[string]any
		decodeBody(t, res, &lBody)
		l, _ := lBody["listing"].(map[string]any)
		if l["status"] != "sold" {
			t.Fatalf("expected listing %s status sold, got %v", id, l["status"])
		}
	}

	// 4. Verify Cart is Empty
	res, err = testApp.Test(authReq("GET", "/api/v1/cart", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	var cartBody map[string]any
	decodeBody(t, res, &cartBody)
	cartItems, _ := cartBody["items"].([]any)
	if len(cartItems) != 0 {
		t.Fatalf("expected 0 cart items, got %d", len(cartItems))
	}

	// 5. Verify Seller received a notification
	// In this test, buyer and seller are the same (not ideal but works for trigger check)
	res, err = testApp.Test(authReq("GET", "/api/v1/notifications", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	var notifBody map[string]any
	decodeBody(t, res, &notifBody)
	notifs, _ := notifBody["notifications"].([]any)
	if len(notifs) < 2 {
		t.Fatalf("expected at least 2 notifications for seller, got %d", len(notifs))
	}
}

func TestOrder_CheckoutUnavailable(t *testing.T) {
	email, pass := "order_fail@test.com", "Str0ngP@ssword!"
	insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	l1 := createListing(t, token, "Going to be sold")
	addToCart(t, token, l1)

	// Manually mark listing as sold
	testDB.MustExec(`UPDATE book_listings SET status = 'sold' WHERE id = $1`, l1)

	// Attempt checkout
	res, err := testApp.Test(authReq("POST", "/api/v1/orders", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusUnprocessableEntity)
}
