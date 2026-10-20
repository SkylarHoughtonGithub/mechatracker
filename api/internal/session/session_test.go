package session

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testKey = []byte("test-signing-key-must-be-at-least-32-bytes-long")

func TestMintRequiresSubject(t *testing.T) {
	_, err := Mint(testKey, Claims{}, time.Hour)
	if err == nil {
		t.Fatal("expected error when subject is empty")
	}
}

func TestMintParseRoundTrip(t *testing.T) {
	in := Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "google-sub-123"},
		Email:            "alice@example.com",
		EmailVerified:    true,
		Name:             "Alice",
		UserID:           "00000000-0000-0000-0000-000000000001",
		OrgID:            "00000000-0000-0000-0000-000000000002",
		Role:             "owner",
	}
	tok, err := Mint(testKey, in, time.Hour)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	out, err := Parse(testKey, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out.Subject != in.Subject || out.Email != in.Email || out.UserID != in.UserID || out.OrgID != in.OrgID || out.Role != in.Role {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", out, in)
	}
	if out.Issuer != Issuer {
		t.Fatalf("issuer: got %q, want %q", out.Issuer, Issuer)
	}
	if out.ExpiresAt == nil || out.ExpiresAt.Time.Before(time.Now()) {
		t.Fatal("expiresAt not set or already past")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	tok, err := Mint(testKey, Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "s"},
	}, -time.Minute)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if _, err := Parse(testKey, tok); err == nil {
		t.Fatal("expected parse to reject expired token")
	}
}

func TestParseRejectsWrongKey(t *testing.T) {
	tok, err := Mint(testKey, Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "s"},
	}, time.Hour)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	otherKey := []byte("different-signing-key-also-32-bytes-long----")
	if _, err := Parse(otherKey, tok); err == nil {
		t.Fatal("expected parse to reject wrong key")
	}
}

func TestParseRejectsWrongIssuer(t *testing.T) {
	// Build a token with a different issuer by signing it directly.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "s",
			Issuer:    "not-mechatracker",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	raw, err := tok.SignedString(testKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := Parse(testKey, raw); err == nil {
		t.Fatal("expected parse to reject wrong issuer")
	}
}

func TestCookieSetAndClear(t *testing.T) {
	c := Cookie("abc", 10*time.Minute, "", false)
	if c.Name != CookieName || c.Value != "abc" || !c.HttpOnly {
		t.Fatalf("set cookie wrong shape: %+v", c)
	}
	if c.MaxAge <= 0 {
		t.Fatalf("set cookie should have positive MaxAge, got %d", c.MaxAge)
	}

	clr := Cookie("", 0, "", true)
	if clr.MaxAge >= 0 {
		t.Fatalf("clear cookie should have MaxAge<0, got %d", clr.MaxAge)
	}
	if clr.Secure != true {
		t.Fatal("clear cookie should honor secure flag")
	}
	if !strings.EqualFold(clr.Name, CookieName) {
		t.Fatalf("clear cookie name: %q", clr.Name)
	}
}
