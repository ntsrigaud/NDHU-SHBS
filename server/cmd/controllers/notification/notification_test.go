package notification_test

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
		tcpostgres.WithDatabase("shbs_test_notification"),
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
	notification.Mount(api, testDB)

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

func TestNotifications_Flow(t *testing.T) {
	email, pass := "notif_test@test.com", "Str0ngP@ssword!"
	userID := insertVerifiedUser(t, email, pass)
	token := loginAndGetToken(t, email, pass)

	// 1. Manually insert a notification
	notifID := uuid.New()
	testDB.MustExec(`
		INSERT INTO notifications (id, user_id, type, payload, is_read)
		VALUES ($1, $2, $3, $4, false)`,
		notifID, userID, "new_message", `{"text": "hello"}`,
	)

	// 2. List notifications
	res, err := testApp.Test(authReq("GET", "/api/v1/notifications", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	var listBody map[string]any
	decodeBody(t, res, &listBody)
	notifs, _ := listBody["notifications"].([]any)
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}

	n, _ := notifs[0].(map[string]any)
	if n["id"] != notifID.String() {
		t.Fatalf("expected notif id %s, got %v", notifID.String(), n["id"])
	}
	if n["is_read"] != false {
		t.Fatal("expected is_read to be false")
	}

	// 3. Mark as read
	res, err = testApp.Test(authReq("PATCH", "/api/v1/notifications/"+notifID.String(), nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)

	// 4. Verify mark as read
	res, err = testApp.Test(authReq("GET", "/api/v1/notifications", nil, token), -1)
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, res, http.StatusOK)
	decodeBody(t, res, &listBody)
	notifs, _ = listBody["notifications"].([]any)
	n, _ = notifs[0].(map[string]any)
	if n["is_read"] != true {
		t.Fatal("expected is_read to be true")
	}
}
