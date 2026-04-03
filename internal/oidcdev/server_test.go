package oidcdev

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Y4shin/open-caucus/internal/oauth"
)

func TestRun_EmitsGroupsClaimFromYAML(t *testing.T) {
	port := freeTCPPort(t)
	issuer := fmt.Sprintf("http://127.0.0.1:%d", port)
	redirectURL := "http://127.0.0.1:19097/oauth/callback"

	tmpDir := t.TempDir()
	usersFile := filepath.Join(tmpDir, "users.yaml")
	usersYAML := `users:
  - id: "id1"
    username: "alice"
    password: "alice"
    first_name: "Alice"
    last_name: "Example"
    email: "alice@example.local"
    email_verified: true
    preferred_language: "en"
    is_admin: false
    groups:
      - "committee-a"
      - "committee-a-chair"
      - "ca-admin"
`
	if err := os.WriteFile(usersFile, []byte(usersYAML), 0o600); err != nil {
		t.Fatalf("write users yaml: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, Config{
			IssuerURL:    issuer,
			ClientID:     "dev-client",
			ClientSecret: "dev-secret",
			RedirectURL:  redirectURL,
			UsersFile:    usersFile,
			GroupsClaim:  "groups",
		})
	}()
	waitForServer(t, issuer+"/.well-known/openid-configuration")

	oauthSvc, err := oauth.New(context.Background(), oauth.Config{
		Enabled:        true,
		IssuerURL:      issuer,
		ClientID:       "dev-client",
		ClientSecret:   "dev-secret",
		RedirectURL:    redirectURL,
		Scopes:         []string{"openid", "profile", "email"},
		GroupsClaim:    "groups",
		UsernameClaims: []string{"preferred_username", "email", "sub"},
		FullNameClaims: []string{"name", "preferred_username", "email"},
		StateTTL:       time.Minute,
	}, []byte("oidcdev-test-secret-1234567890"))
	if err != nil {
		t.Fatalf("create oauth service: %v", err)
	}

	authURL, stateCookie, err := oauthSvc.Start("user")
	if err != nil {
		t.Fatalf("oauth start: %v", err)
	}
	callbackURL := completeOIDCLoginForLocalDev(t, authURL, "alice", "alice")

	req := httptest.NewRequest(http.MethodGet, callbackURL, nil)
	req.AddCookie(stateCookie)
	result, err := oauthSvc.HandleCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("oauth callback: %v", err)
	}
	if !contains(result.Principal.Groups, "committee-a") || !contains(result.Principal.Groups, "committee-a-chair") {
		t.Fatalf("expected groups claim from YAML, got=%v", result.Principal.Groups)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("oidcdev server returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for oidcdev shutdown")
	}
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen tcp :0: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitForServer(t *testing.T, target string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(target)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server not ready: %s", target)
}

func completeOIDCLoginForLocalDev(t *testing.T, authURL, username, password string) string {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(authURL)
	if err != nil {
		t.Fatalf("get authorize URL: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected first authorize response status 302, got=%d", resp.StatusCode)
	}
	loginURL := resolveLocation(resp.Request.URL, resp.Header.Get("Location"))
	authReqID := mustQueryValue(t, loginURL, "authRequestID")

	getLoginResp, err := client.Get(loginURL)
	if err != nil {
		t.Fatalf("get login page: %v", err)
	}
	getLoginResp.Body.Close()
	if getLoginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected login page status 200, got=%d", getLoginResp.StatusCode)
	}

	form := url.Values{
		"username": {username},
		"password": {password},
		"id":       {authReqID},
	}
	postResp, err := client.PostForm(loginURL, form)
	if err != nil {
		t.Fatalf("post login form: %v", err)
	}
	postResp.Body.Close()
	if postResp.StatusCode != http.StatusFound {
		t.Fatalf("expected login submit status 302, got=%d", postResp.StatusCode)
	}

	oidcCallbackURL := resolveLocation(postResp.Request.URL, postResp.Header.Get("Location"))
	authorizeCallbackResp, err := client.Get(oidcCallbackURL)
	if err != nil {
		t.Fatalf("get authorize callback: %v", err)
	}
	authorizeCallbackResp.Body.Close()
	if authorizeCallbackResp.StatusCode != http.StatusFound {
		t.Fatalf("expected authorize callback status 302, got=%d", authorizeCallbackResp.StatusCode)
	}
	callbackURL := resolveLocation(authorizeCallbackResp.Request.URL, authorizeCallbackResp.Header.Get("Location"))
	if !strings.Contains(callbackURL, "code=") || !strings.Contains(callbackURL, "state=") {
		t.Fatalf("expected callback redirect with code/state, got=%q", callbackURL)
	}
	return callbackURL
}

func resolveLocation(base *url.URL, location string) string {
	target, err := url.Parse(location)
	if err != nil {
		return location
	}
	return base.ResolveReference(target).String()
}

func mustQueryValue(t *testing.T, rawURL, key string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse URL %q: %v", rawURL, err)
	}
	value := strings.TrimSpace(u.Query().Get(key))
	if value == "" {
		t.Fatalf("missing query param %q in URL %q", key, rawURL)
	}
	return value
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
