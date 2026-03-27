package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSPAHandlerServesBuiltAssets(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/_app/env.js", nil)
	rec := httptest.NewRecorder()

	NewSPAHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected asset status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("expected javascript content type, got %q", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "env") {
		t.Fatalf("expected asset body, got %q", body)
	}
}

func TestSPAHandlerFallsBackToIndexForRoutes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/login", nil)
	rec := httptest.NewRecorder()

	NewSPAHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected route status 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected html content type, got %q", got)
	}
	if body := rec.Body.String(); !strings.Contains(body, "__sveltekit") {
		t.Fatalf("expected SPA index body, got %q", body)
	}
}
