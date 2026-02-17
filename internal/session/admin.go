package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// AdminSessionManager handles admin authentication sessions
// Uses a separate cookie from regular user sessions
type AdminSessionManager struct {
	secret []byte
}

// NewAdminSessionManager creates a new admin session manager
func NewAdminSessionManager(secret []byte) *AdminSessionManager {
	if len(secret) < 32 {
		panic("admin session secret must be at least 32 bytes")
	}
	return &AdminSessionManager{
		secret: secret,
	}
}

// CreateAdminSession creates a signed admin session token
func (m *AdminSessionManager) CreateAdminSession() string {
	// Generate a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		panic(fmt.Sprintf("failed to generate admin session token: %v", err))
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Sign the token
	return m.signToken(token)
}

// ValidateAdminSession verifies an admin session token
func (m *AdminSessionManager) ValidateAdminSession(signedToken string) bool {
	parts := strings.Split(signedToken, ".")
	if len(parts) != 2 {
		return false
	}

	token := parts[0]
	providedSignature := parts[1]

	// Compute expected signature
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(token))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(providedSignature), []byte(expectedSignature))
}

// CreateAdminCookie creates an HTTP cookie for admin sessions
func (m *AdminSessionManager) CreateAdminCookie(signedToken string) *http.Cookie {
	return &http.Cookie{
		Name:     "admin_session",
		Value:    signedToken,
		Path:     "/admin",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // Enable in production with HTTPS
	}
}

// ClearAdminCookie creates a cookie that clears the admin session
func (m *AdminSessionManager) ClearAdminCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "admin_session",
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		HttpOnly: true,
	}
}

// signToken creates an HMAC signature for the token
func (m *AdminSessionManager) signToken(token string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(token))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", token, signature)
}

// Context key for admin authentication
type adminContextKey int

const adminAuthContextKey adminContextKey = 0

// WithAdminAuth marks the context as authenticated for admin
func WithAdminAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, adminAuthContextKey, true)
}

// IsAdminAuthenticated checks if the context has admin authentication
func IsAdminAuthenticated(ctx context.Context) bool {
	auth, ok := ctx.Value(adminAuthContextKey).(bool)
	return ok && auth
}
