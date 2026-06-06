package message_test

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
	"shbs-server/cmd/controllers/message"
	"shbs-server/cmd/controllers/notification"
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
		tcpostgres.WithDatabase("shbs_test_message"),
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
	testApp.Get("/api/v1/messages/unread-count", authMW, func(c *fiber.Ctx) error { return message.GetUnreadMessageCount(testDB, c) })
	testApp.Get("/api/v1/messages/conversations", authMW, func(c *fiber.Ctx) error { return message.GetConversations(testDB, c) })
	testApp.Patch("/api/v1/messages/:id/read", authMW, func(c *fiber.Ctx) error { return message.MarkMessageAsRead(testDB, c) })
	testApp.Get("/api/v1/listings/:listingId/messages", authMW, func(c *fiber.Ctx) error { return message.GetMessages(testDB, c) })
	testApp.Post("/api/v1/listings/:listingId/messages", authMW, func(c *fiber.Ctx) error { return message.SendMessage(testDB, c) })
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

func insertVerifiedUser(t *testing.T, email, plainPassword string) string {
	t.Helper()
	hash, err := util.HashPassword(plainPassword)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	_, err = testDB.Exec(`
		INSERT INTO users (id, name, email, password_hash, email_verified)
		VALUES ($1, $2, $3, $4, true)`,
		id, strings.Split(email, "@")[0], email, hash,
	)
	if err != nil {
		t.Fatal("insertVerifiedUser:", err)
	}
	return id.String()
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

func TestMessaging_Flow(t *testing.T) {
	sellerEmail, sellerPass := "seller_msg@test.com", "Str0ngP@ssword!"
	_ = insertVerifiedUser(t, sellerEmail, sellerPass)
	sellerToken := loginAndGetToken(t, sellerEmail, sellerPass)

	buyerEmail, buyerPass := "buyer_msg@test.com", "Str0ngP@ssword!"
	buyerID := insertVerifiedUser(t, buyerEmail, buyerPass)
	buyerToken := loginAndGetToken(t, buyerEmail, buyerPass)

	listingID := createListing(t, sellerToken, "Message Book")

	// 1. Buyer sends message to Seller
	res, err := testApp.Test(authReq("POST", "/api/v1/messages", map[string]string{
		"listing_id": listingID,
		"body":       "Hi, is this available?",
	}, buyerToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	// 2. Seller lists messages (convo with Buyer)
	res, err = testApp.Test(authReq("GET", "/api/v1/messages?listing_id="+listingID+"&other_user_id="+buyerID, nil, sellerToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var listBody map[string]any
	decodeBody(t, res, &listBody)
	msgs, _ := listBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	m, _ := msgs[0].(map[string]any)
	if m["body"] != "Hi, is this available?" {
		t.Fatalf("unexpected message body: %v", m["body"])
	}

	// 3. Seller replies
	res, err = testApp.Test(authReq("POST", "/api/v1/messages", map[string]string{
		"listing_id":  listingID,
		"receiver_id": buyerID,
		"body":        "Yes, it is!",
	}, sellerToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusCreated)

	// 4. Buyer lists messages
	res, err = testApp.Test(authReq("GET", "/api/v1/messages?listing_id="+listingID, nil, buyerToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	decodeBody(t, res, &listBody)
	msgs, _ = listBody["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages in convo, got %d", len(msgs))
	}

	// 5. Verify Receiver (Seller) received a notification for the first message
	res, err = testApp.Test(authReq("GET", "/api/v1/notifications", nil, sellerToken), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	var notifBody map[string]any
	decodeBody(t, res, &notifBody)
	notifs, _ := notifBody["notifications"].([]any)
	if len(notifs) == 0 {
		t.Fatal("expected at least 1 notification for seller")
	}
}
