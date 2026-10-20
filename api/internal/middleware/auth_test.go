package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/middleware"
)

func TestAuthDevBypassSetsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Env:           "development",
		DevAuthBypass: true,
	}

	r := gin.New()
	r.Use(middleware.Auth(cfg, zap.NewNop()))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetString(middleware.KeyUserID),
			"org_id":     c.GetString(middleware.KeyOrgID),
			"role":       c.GetString(middleware.KeyRole),
			"google_sub": c.GetString(middleware.KeyGoogleSub),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`"user_id":"00000000-0000-0000-0000-000000000002"`,
		`"org_id":"00000000-0000-0000-0000-000000000003"`,
		`"role":"owner"`,
		`"google_sub":"dev_google_sub_local"`,
	} {
		if !containsString(body, want) {
			t.Errorf("response missing %s; got %s", want, body)
		}
	}
}

func TestAuthRejectsMissingCredential(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Env:               "production",
		GoogleClientID:    "test-client-id",
		SessionSigningKey: "test-signing-key-must-be-at-least-32-bytes-long",
	}

	r := gin.New()
	r.Use(middleware.Auth(cfg, zap.NewNop()))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
