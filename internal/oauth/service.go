package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const stateCookieName = "oauth_state"

// Config configures OAuth/OIDC authentication.
type Config struct {
	Enabled bool

	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string

	GroupsClaim    string
	UsernameClaims []string
	FullNameClaims []string

	StateTTL time.Duration
}

// Principal is the normalized identity extracted from an ID token.
type Principal struct {
	Issuer   string
	Subject  string
	Username string
	FullName string
	Email    string
	Groups   []string
}

type callbackState struct {
	Target       string `json:"target"`
	State        string `json:"state"`
	Nonce        string `json:"nonce"`
	CodeVerifier string `json:"code_verifier"`
	ExpiresAt    int64  `json:"exp"`
}

// CallbackResult contains verified OAuth callback state and identity.
type CallbackResult struct {
	Target    string
	Principal Principal
}

// Service performs OAuth/OIDC operations.
type Service struct {
	enabled bool

	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier

	groupsClaim    string
	usernameClaims []string
	fullNameClaims []string
	stateTTL       time.Duration
	stateSecret    []byte
}

// New creates a new OIDC-backed OAuth service.
func New(ctx context.Context, cfg Config, stateSecret []byte) (*Service, error) {
	svc := &Service{
		enabled: cfg.Enabled,
	}
	if !cfg.Enabled {
		return svc, nil
	}
	if len(stateSecret) < 16 {
		return nil, fmt.Errorf("oauth state secret must be at least 16 bytes")
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc provider discovery: %w", err)
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	groupsClaim := strings.TrimSpace(cfg.GroupsClaim)
	if groupsClaim == "" {
		groupsClaim = "groups"
	}
	usernameClaims := cfg.UsernameClaims
	if len(usernameClaims) == 0 {
		usernameClaims = []string{"preferred_username", "email", "sub"}
	}
	fullNameClaims := cfg.FullNameClaims
	if len(fullNameClaims) == 0 {
		fullNameClaims = []string{"name", "preferred_username", "email"}
	}
	stateTTL := cfg.StateTTL
	if stateTTL <= 0 {
		stateTTL = 5 * time.Minute
	}
	return &Service{
		enabled: true,
		oauth2Config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		},
		verifier:       provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		groupsClaim:    groupsClaim,
		usernameClaims: usernameClaims,
		fullNameClaims: fullNameClaims,
		stateTTL:       stateTTL,
		stateSecret:    stateSecret,
	}, nil
}

// Enabled returns whether OAuth is enabled.
func (s *Service) Enabled() bool {
	return s != nil && s.enabled
}

// Start creates auth URL and state cookie for a login target.
func (s *Service) Start(target string) (string, *http.Cookie, error) {
	if !s.Enabled() {
		return "", nil, fmt.Errorf("oauth provider disabled")
	}
	if target != "admin" {
		target = "user"
	}
	state, err := randomToken(32)
	if err != nil {
		return "", nil, fmt.Errorf("generate oauth state: %w", err)
	}
	nonce, err := randomToken(32)
	if err != nil {
		return "", nil, fmt.Errorf("generate oauth nonce: %w", err)
	}
	codeVerifier, err := randomToken(48)
	if err != nil {
		return "", nil, fmt.Errorf("generate oauth code verifier: %w", err)
	}

	p := callbackState{
		Target:       target,
		State:        state,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(s.stateTTL).Unix(),
	}
	stateCookieValue, err := s.encodeStateCookie(p)
	if err != nil {
		return "", nil, err
	}
	cookie := &http.Cookie{
		Name:     stateCookieName,
		Value:    stateCookieValue,
		Path:     "/",
		MaxAge:   int(s.stateTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	authURL := s.oauth2Config.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", pkceS256(codeVerifier)),
	)
	return authURL, cookie, nil
}

// ClearStateCookie returns an expired state cookie to clear browser state.
func (s *Service) ClearStateCookie() *http.Cookie {
	return &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// HandleCallback verifies callback state and ID token and returns principal details.
func (s *Service) HandleCallback(ctx context.Context, r *http.Request) (*CallbackResult, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("oauth provider disabled")
	}
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if state == "" || code == "" {
		return nil, fmt.Errorf("missing oauth callback parameters")
	}
	cookie, err := r.Cookie(stateCookieName)
	if err != nil || cookie.Value == "" {
		return nil, fmt.Errorf("missing oauth state cookie")
	}
	payload, err := s.decodeStateCookie(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth state cookie: %w", err)
	}
	if time.Now().Unix() > payload.ExpiresAt {
		return nil, fmt.Errorf("oauth state expired")
	}
	if payload.State != state {
		return nil, fmt.Errorf("oauth state mismatch")
	}

	token, err := s.oauth2Config.Exchange(
		ctx,
		code,
		oauth2.SetAuthURLParam("code_verifier", payload.CodeVerifier),
	)
	if err != nil {
		return nil, fmt.Errorf("oauth code exchange failed: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, fmt.Errorf("missing id_token in token response")
	}
	idToken, err := s.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("id token verification failed: %w", err)
	}

	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode id token claims: %w", err)
	}

	tokenNonce, _ := claims["nonce"].(string)
	if tokenNonce == "" || tokenNonce != payload.Nonce {
		return nil, fmt.Errorf("nonce mismatch")
	}

	principal := Principal{
		Issuer:  idToken.Issuer,
		Subject: idToken.Subject,
		Email:   stringClaim(claims, "email"),
		Groups:  groupsClaim(claims, s.groupsClaim),
	}
	principal.Username = firstAvailableClaim(claims, s.usernameClaims)
	principal.FullName = firstAvailableClaim(claims, s.fullNameClaims)
	if principal.Username == "" {
		principal.Username = principal.Subject
	}
	if principal.FullName == "" {
		principal.FullName = principal.Username
	}

	return &CallbackResult{
		Target:    payload.Target,
		Principal: principal,
	}, nil
}

func stringClaim(claims map[string]any, key string) string {
	raw, ok := claims[key]
	if !ok {
		return ""
	}
	v, _ := raw.(string)
	return strings.TrimSpace(v)
}

func firstAvailableClaim(claims map[string]any, chain []string) string {
	for _, name := range chain {
		if v := stringClaim(claims, name); v != "" {
			return v
		}
	}
	return ""
}

func groupsClaim(claims map[string]any, claimName string) []string {
	raw, ok := claims[claimName]
	if !ok {
		slog.Debug("oauth groups claim: claim not present in token", "claim_name", claimName)
		return nil
	}
	var groups []string
	switch typed := raw.(type) {
	case []any:
		groups = make([]string, 0, len(typed))
		for _, item := range typed {
			if v, ok := item.(string); ok && strings.TrimSpace(v) != "" {
				groups = append(groups, strings.TrimSpace(v))
			}
		}
	case []string:
		groups = make([]string, 0, len(typed))
		for _, item := range typed {
			if strings.TrimSpace(item) != "" {
				groups = append(groups, strings.TrimSpace(item))
			}
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			slog.Debug("oauth groups claim: claim is empty string", "claim_name", claimName)
			return nil
		}
		groups = []string{strings.TrimSpace(typed)}
	default:
		slog.Debug("oauth groups claim: unexpected claim type", "claim_name", claimName, "raw_type", fmt.Sprintf("%T", raw))
		return nil
	}
	slog.Debug("oauth groups claim: extracted groups", "claim_name", claimName, "groups", groups, "count", len(groups))
	return groups
}

func (s *Service) encodeStateCookie(p callbackState) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("encode state payload: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, s.stateSecret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return "", fmt.Errorf("sign state payload: %w", err)
	}
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig, nil
}

func (s *Service) decodeStateCookie(value string) (*callbackState, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state cookie format")
	}
	payload := parts[0]
	sig := parts[1]

	mac := hmac.New(sha256.New, s.stateSecret)
	if _, err := mac.Write([]byte(payload)); err != nil {
		return nil, fmt.Errorf("sign state payload: %w", err)
	}
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return nil, fmt.Errorf("state cookie signature mismatch")
	}

	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode state payload: %w", err)
	}
	var p callbackState
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal state payload: %w", err)
	}
	return &p, nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

