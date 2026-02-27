package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Y4shin/conference-tool/internal/config"
)

func TestOAuthStart_DisabledProviderReturnsNotFound(t *testing.T) {
	h := &Handler{
		AuthConfig: &config.AuthConfig{
			OAuthEnabled: false,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/oauth/start?target=user", nil)
	rec := httptest.NewRecorder()

	if err := h.OAuthStart(rec, req); err != nil {
		t.Fatalf("OAuthStart returned unexpected error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 when oauth disabled, got=%d", rec.Code)
	}
}

func TestOAuthCallback_DisabledProviderReturnsNotFound(t *testing.T) {
	h := &Handler{
		AuthConfig: &config.AuthConfig{
			OAuthEnabled: false,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/oauth/callback?code=test&state=test", nil)
	rec := httptest.NewRecorder()

	if err := h.OAuthCallback(rec, req); err != nil {
		t.Fatalf("OAuthCallback returned unexpected error: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 when oauth disabled, got=%d", rec.Code)
	}
}
