package auth

import (
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func useSecureSessionCookie(c *fiber.Ctx) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true") {
		return true
	}

	if strings.EqualFold(c.Protocol(), "https") {
		return true
	}

	forwardedProto := strings.TrimSpace(strings.Split(c.Get("X-Forwarded-Proto"), ",")[0])
	return strings.EqualFold(forwardedProto, "https")
}

func setSessionCookie(c *fiber.Ctx, token string, expires time.Time) {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HTTPOnly: true,
		Secure:   useSecureSessionCookie(c),
		SameSite: "Lax",
	})
}

func clearSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   useSecureSessionCookie(c),
		SameSite: "Lax",
	})
}