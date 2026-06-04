package auth

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"shbs-server/pkg/model"
	"shbs-server/pkg/util"
)

// ─── CAS XML types ──────────────────────────────────────────────────────────

// casServiceResponse is the envelope returned by CAS /cas/serviceValidate.
type casServiceResponse struct {
	Success *casAuthSuccess `xml:"authenticationSuccess"`
	Failure *casAuthFailure `xml:"authenticationFailure"`
}

type casAuthSuccess struct {
	User       string        `xml:"user"`
	Attributes casAttributes `xml:"attributes"`
}

type casAttributes struct {
	Name  string `xml:"cn"`
	Email string `xml:"mail"`
}

type casAuthFailure struct {
	Code    string `xml:"code,attr"`
	Message string `xml:",chardata"`
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// ssoCallbackURL returns the full URL CAS must redirect back to.
func ssoCallbackURL() string {
	return strings.TrimRight(os.Getenv("API_BASE_URL"), "/") + "/api/v1/auth/sso/callback"
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// HandleSSOLogin redirects the browser to the NDHU CAS login page.
// Set NDHU_CAS_BASE_URL (e.g. https://cas.ndhu.edu.tw) in the environment.
func HandleSSOLogin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		casBase := strings.TrimRight(os.Getenv("NDHU_CAS_BASE_URL"), "/")
		if casBase == "" {
			return fiber.NewError(fiber.StatusServiceUnavailable, "SSO is not configured")
		}
		loginURL := fmt.Sprintf("%s/cas/login?service=%s",
			casBase, url.QueryEscape(ssoCallbackURL()))
		return c.Redirect(loginURL, fiber.StatusFound)
	}
}

// HandleSSOCallback validates the CAS ticket, upserts the user, and issues a JWT.
//
// Dev mode: set NDHU_SSO_MOCK=true and present tickets of the form
// "dev-ticket-<cas_id>" to bypass the real CAS server.
func HandleSSOCallback(db *sqlx.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ticket := c.Query("ticket")
		if ticket == "" {
			return fiber.NewError(fiber.StatusBadRequest, "missing CAS ticket")
		}

		var casID, casName, casEmail string

		if os.Getenv("NDHU_SSO_MOCK") == "true" {
			const prefix = "dev-ticket-"
			if !strings.HasPrefix(ticket, prefix) {
				return fiber.NewError(fiber.StatusUnauthorized, "invalid mock ticket")
			}
			casID = ticket[len(prefix):]
			casName = casID
			casEmail = casID + "@gm.ndhu.edu.tw"
		} else {
			var err error
			casID, casName, casEmail, err = validateCASTicket(ticket)
			if err != nil {
				return fiber.NewError(fiber.StatusUnauthorized, "CAS ticket validation failed")
			}
		}

		// Fill in defaults when CAS attributes are absent.
		if casEmail == "" {
			casEmail = casID + "@gm.ndhu.edu.tw"
		}
		if casName == "" {
			casName = casID
		}

		user, err := upsertCASUser(db, casID, casName, casEmail)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not sign in via SSO")
		}

		token, expiresAt, err := util.GenerateJWT(user.ID, user.IsAdmin)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not generate token")
		}

		c.Cookie(&fiber.Cookie{
			Name:     "jwt",
			Value:    token,
			Expires:  expiresAt,
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		frontendURL := strings.TrimRight(os.Getenv("FRONTEND_BASE_URL"), "/")
		return c.Redirect(frontendURL+"/", fiber.StatusFound)
	}
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// validateCASTicket calls the CAS serviceValidate endpoint and parses the XML
// response to extract the authenticated user's identity attributes.
func validateCASTicket(ticket string) (casID, casName, casEmail string, err error) {
	casBase := strings.TrimRight(os.Getenv("NDHU_CAS_BASE_URL"), "/")
	validateURL := fmt.Sprintf("%s/cas/serviceValidate?service=%s&ticket=%s",
		casBase,
		url.QueryEscape(ssoCallbackURL()),
		url.QueryEscape(ticket),
	)

	resp, err := http.Get(validateURL) //nolint:noctx
	if err != nil {
		return "", "", "", fmt.Errorf("CAS HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("read CAS response: %w", err)
	}

	var cas casServiceResponse
	if xmlErr := xml.Unmarshal(body, &cas); xmlErr != nil {
		return "", "", "", fmt.Errorf("parse CAS XML: %w", xmlErr)
	}

	if cas.Failure != nil {
		return "", "", "", fmt.Errorf("CAS error %s: %s",
			cas.Failure.Code, strings.TrimSpace(cas.Failure.Message))
	}
	if cas.Success == nil || cas.Success.User == "" {
		return "", "", "", errors.New("CAS returned no authenticated user")
	}

	return cas.Success.User,
		cas.Success.Attributes.Name,
		cas.Success.Attributes.Email,
		nil
}

// upsertCASUser finds or creates a user for the given CAS identity.
//
// Resolution order:
//  1. Existing user with matching cas_id  → touch updated_at and return.
//  2. Existing user with matching email but no cas_id → link the cas_id.
//  3. No match → insert a new, pre-verified user.
func upsertCASUser(db *sqlx.DB, casID, name, email string) (model.User, error) {
	var user model.User

	// 1. Already signed in via CAS before.
	err := db.Get(&user, `SELECT * FROM users WHERE cas_id = $1`, casID)
	if err == nil {
		_, _ = db.Exec(`UPDATE users SET updated_at = NOW() WHERE id = $1`, user.ID)
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("lookup by cas_id: %w", err)
	}

	// 2. Local account with same email exists — link CAS to it.
	err = db.Get(&user, `SELECT * FROM users WHERE email = $1`, email)
	if err == nil {
		err = db.QueryRowx(
			`UPDATE users SET cas_id = $1, updated_at = NOW() WHERE id = $2 RETURNING *`,
			casID, user.ID,
		).StructScan(&user)
		return user, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("lookup by email: %w", err)
	}

	// 3. Brand-new user — email is pre-verified because CAS is authoritative.
	err = db.QueryRowx(`
		INSERT INTO users (id, name, email, cas_id, email_verified)
		VALUES ($1, $2, $3, $4, true)
		RETURNING *`,
		uuid.New(), name, email, casID,
	).StructScan(&user)
	return user, err
}
