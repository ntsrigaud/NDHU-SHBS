package util

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
)

// Claims is the JWT payload structure stored in every signed token.
type Claims struct {
	UserID  uuid.UUID `json:"user_id"`
	IsAdmin bool      `json:"is_admin"`
	jwt.RegisteredClaims
}

// jwtSecret returns the signing secret from the environment. It is resolved at
// call time (not at package init) so that tests can set the variable before
// calling JWT functions.
func jwtSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// jwtExpiry returns the token lifetime from JWT_EXPIRY_HOURS (default: 24h).
func jwtExpiry() time.Duration {
	hours, err := strconv.Atoi(os.Getenv("JWT_EXPIRY_HOURS"))
	if err != nil || hours <= 0 {
		hours = 24
	}
	return time.Duration(hours) * time.Hour
}

// GenerateJWT creates a signed HS256 JWT for the given user.
func GenerateJWT(userID uuid.UUID, isAdmin bool) (string, time.Time, error) {
	expiresAt := time.Now().Add(jwtExpiry())
	claims := &Claims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret())
	return signed, expiresAt, err
}

// ParseJWT validates the token string and returns its claims.
func ParseJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret(), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ExtractUserIDFromJwtToken reads the Authorization header or the jwt cookie
// from the Fiber context, validates the token, and returns the user UUID.
func ExtractUserIDFromJwtToken(c *fiber.Ctx) (uuid.UUID, error) {
	tokenString := extractRawToken(c)
	if tokenString == "" {
		return uuid.Nil, errors.New("missing or malformed token")
	}
	claims, err := ParseJWT(tokenString)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

// ExtractClaims is the full-claims variant of ExtractUserIDFromJwtToken.
func ExtractClaims(c *fiber.Ctx) (*Claims, error) {
	tokenString := extractRawToken(c)
	if tokenString == "" {
		return nil, errors.New("missing or malformed token")
	}
	return ParseJWT(tokenString)
}

// extractRawToken prefers the httpOnly jwt cookie, then falls back to the
// Authorization: Bearer <token> header.
func extractRawToken(c *fiber.Ctx) string {
	if cookie := c.Cookies("jwt"); cookie != "" {
		return cookie
	}
	auth := c.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
