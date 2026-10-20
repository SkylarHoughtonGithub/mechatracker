// Package session mints and parses the HS256 JWT the API uses to track a
// signed-in user. The token lives in an httpOnly cookie and carries both the
// Google identity (sub/email/name) and, once the user has onboarded, the
// internal users.id and org_id.
package session

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	CookieName = "mt_session"
	Issuer     = "mechatracker"
)

// Claims is the JWT payload. google_sub lives in the standard Subject slot.
// UserID and OrgID are empty strings for pre-onboarding sessions (user has
// signed in with Google but hasn't created or joined an org yet).
type Claims struct {
	jwt.RegisteredClaims
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	PictureURL    string `json:"picture,omitempty"`
	UserID        string `json:"user_id,omitempty"`
	OrgID         string `json:"org_id,omitempty"`
	Role          string `json:"role,omitempty"`
}

// Mint signs a new session JWT. ttl should come from Config.SessionTTL.
func Mint(key []byte, c Claims, ttl time.Duration) (string, error) {
	now := time.Now()
	c.Issuer = Issuer
	c.IssuedAt = jwt.NewNumericDate(now)
	c.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	if c.Subject == "" {
		return "", fmt.Errorf("session: subject (google_sub) is required")
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &c)
	return tok.SignedString(key)
}

// Parse verifies the signature + expiry and returns the claims.
func Parse(key []byte, raw string) (*Claims, error) {
	var c Claims
	tok, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	}, jwt.WithExpirationRequired(), jwt.WithIssuer(Issuer))
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, fmt.Errorf("session: invalid token")
	}
	return &c, nil
}

// Cookie builds the httpOnly cookie that carries the session JWT. Pass an
// empty token with MaxAge=-1 to clear.
func Cookie(token string, ttl time.Duration, domain string, secure bool) *http.Cookie {
	c := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	if token == "" {
		c.MaxAge = -1
		c.Expires = time.Unix(0, 0)
	} else {
		c.MaxAge = int(ttl.Seconds())
		c.Expires = time.Now().Add(ttl)
	}
	return c
}

// UUIDOrNil parses s as a UUID, returning uuid.Nil if s is empty. Handlers use
// this when reading UserID/OrgID off claims where empty is valid.
func UUIDOrNil(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(s)
}
