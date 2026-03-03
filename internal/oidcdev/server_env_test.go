package oidcdev

import (
	"strings"
	"testing"
)

func TestLoadConfigFromEnv_WithListenAddr(t *testing.T) {
	t.Setenv("OAUTH_ISSUER_URL", "http://127.0.0.1:9096")
	t.Setenv("OAUTH_CLIENT_ID", "dev-client")
	t.Setenv("OAUTH_CLIENT_SECRET", "dev-secret")
	t.Setenv("OAUTH_REDIRECT_URL", "http://127.0.0.1:8080/oauth/callback")
	t.Setenv("OIDC_DEV_LISTEN_ADDR", "0.0.0.0:9096")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}
	if cfg.ListenAddr != "0.0.0.0:9096" {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, "0.0.0.0:9096")
	}
}

func TestLoadConfigFromEnv_InvalidListenAddr(t *testing.T) {
	t.Setenv("OAUTH_ISSUER_URL", "http://127.0.0.1:9096")
	t.Setenv("OAUTH_CLIENT_ID", "dev-client")
	t.Setenv("OAUTH_CLIENT_SECRET", "dev-secret")
	t.Setenv("OAUTH_REDIRECT_URL", "http://127.0.0.1:8080/oauth/callback")
	t.Setenv("OIDC_DEV_LISTEN_ADDR", "9096")

	_, err := LoadConfigFromEnv()
	if err == nil {
		t.Fatalf("expected error for invalid listen addr")
	}
	if !strings.Contains(err.Error(), "OIDC_DEV_LISTEN_ADDR") {
		t.Fatalf("unexpected error: %v", err)
	}
}
