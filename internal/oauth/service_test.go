package oauth

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	oidctest "github.com/Y4shin/open-caucus/internal/testsupport/oidc"
)

func TestServiceHandleCallbackSuccess_ZitadelOP(t *testing.T) {
	callbackURL := "http://127.0.0.1:19091/oauth/callback"
	provider, err := oidctest.Start(oidctest.Config{
		RedirectURL: callbackURL,
	})
	if err != nil {
		t.Fatalf("start oidc provider: %v", err)
	}
	t.Cleanup(provider.Close)

	service, err := New(context.Background(), Config{
		Enabled:        true,
		IssuerURL:      provider.Issuer,
		ClientID:       provider.ClientID,
		ClientSecret:   provider.ClientSecret,
		RedirectURL:    callbackURL,
		Scopes:         []string{"openid", "profile", "email"},
		GroupsClaim:    "aud",
		UsernameClaims: []string{"preferred_username", "email", "sub"},
		FullNameClaims: []string{"name", "preferred_username", "email"},
		StateTTL:       time.Minute,
	}, []byte("oauth-state-secret-1234567890"))
	if err != nil {
		t.Fatalf("create oauth service: %v", err)
	}

	authURL, stateCookie, err := service.Start("admin")
	if err != nil {
		t.Fatalf("oauth start: %v", err)
	}
	redirectedCallbackURL := completeOIDCLogin(t, authURL, provider.Username, provider.Password)

	req := httptest.NewRequest(http.MethodGet, redirectedCallbackURL, nil)
	req.AddCookie(stateCookie)
	result, err := service.HandleCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("oauth callback: %v", err)
	}
	if result.Target != "admin" {
		t.Fatalf("unexpected callback target: got=%q want=%q", result.Target, "admin")
	}
	if result.Principal.Subject == "" {
		t.Fatalf("expected non-empty subject")
	}
	if result.Principal.Username == "" {
		t.Fatalf("expected non-empty username")
	}
	if !containsString(result.Principal.Groups, provider.ClientID) {
		t.Fatalf("expected groups to include client id %q, got=%v", provider.ClientID, result.Principal.Groups)
	}
}

func TestServiceHandleCallbackRejectsReplay_ZitadelOP(t *testing.T) {
	callbackURL := "http://127.0.0.1:19092/oauth/callback"
	provider, err := oidctest.Start(oidctest.Config{
		RedirectURL: callbackURL,
	})
	if err != nil {
		t.Fatalf("start oidc provider: %v", err)
	}
	t.Cleanup(provider.Close)

	service, err := New(context.Background(), Config{
		Enabled:      true,
		IssuerURL:    provider.Issuer,
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		RedirectURL:  callbackURL,
		Scopes:       []string{"openid", "profile", "email"},
		GroupsClaim:  "aud",
		StateTTL:     time.Minute,
	}, []byte("oauth-state-secret-1234567890"))
	if err != nil {
		t.Fatalf("create oauth service: %v", err)
	}

	authURL, stateCookie, err := service.Start("user")
	if err != nil {
		t.Fatalf("oauth start: %v", err)
	}
	redirectedCallbackURL := completeOIDCLogin(t, authURL, provider.Username, provider.Password)

	req1 := httptest.NewRequest(http.MethodGet, redirectedCallbackURL, nil)
	req1.AddCookie(stateCookie)
	if _, err := service.HandleCallback(context.Background(), req1); err != nil {
		t.Fatalf("first oauth callback should succeed: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, redirectedCallbackURL, nil)
	req2.AddCookie(stateCookie)
	if _, err := service.HandleCallback(context.Background(), req2); err == nil {
		t.Fatalf("second oauth callback should fail for replayed authorization code")
	}
}

func TestServiceStartDisabled(t *testing.T) {
	service, err := New(context.Background(), Config{Enabled: false}, []byte("oauth-state-secret-1234567890"))
	if err != nil {
		t.Fatalf("create disabled oauth service: %v", err)
	}
	if _, _, err := service.Start("user"); err == nil {
		t.Fatalf("expected Start to fail when OAuth is disabled")
	}
}

func completeOIDCLogin(t *testing.T, authURL, username, password string) string {
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
	if !strings.Contains(loginURL, "/login/username") {
		t.Fatalf("expected redirect to login page, got=%q", loginURL)
	}

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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
