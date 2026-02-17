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
	"time"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

// Manager handles session creation, validation, and cookie management
type Manager struct {
	store  Store
	secret []byte
}

// NewManager creates a new session manager
func NewManager(store Store, secret []byte) *Manager {
	if len(secret) < 32 {
		panic("session secret must be at least 32 bytes")
	}
	return &Manager{
		store:  store,
		secret: secret,
	}
}

// CreateSession stores session data and returns a signed session ID
func (m *Manager) CreateSession(ctx context.Context, data *SessionData) (string, error) {
	// Generate cryptographically secure session ID
	sessionID := generateSessionID()

	// Convert to model.Session for storage
	modelSession := &model.Session{
		SessionID:   sessionID,
		SessionType: model.SessionType(data.SessionType),
		CreatedAt:   time.Now(),
		ExpiresAt:   data.ExpiresAt,
	}

	// Copy fields based on session type
	if data.IsUserSession() {
		modelSession.UserID = data.UserID
		modelSession.CommitteeSlug = data.CommitteeSlug
		modelSession.Username = data.Username
		modelSession.Role = data.Role
	} else {
		modelSession.AttendeeID = data.AttendeeID
		modelSession.MeetingID = data.MeetingID
		modelSession.FullName = data.FullName
		modelSession.IsChair = data.IsChair
	}

	// Store in database
	if err := m.store.CreateSession(ctx, modelSession); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	// Return signed session ID
	return m.signSessionID(sessionID), nil
}

// GetSession retrieves session data from a signed session ID
func (m *Manager) GetSession(ctx context.Context, signedID string) (*SessionData, error) {
	// Validate signature and extract session ID
	sessionID, valid := m.validateSignedSessionID(signedID)
	if !valid {
		return nil, fmt.Errorf("invalid session signature")
	}

	// Retrieve from database
	modelSession, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if modelSession.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}

	// Convert to SessionData
	return sessionDataFromModel(modelSession), nil
}

// DestroySession removes a session from the store
func (m *Manager) DestroySession(ctx context.Context, signedID string) error {
	// Extract session ID (even if signature is invalid, we want to clean up)
	parts := strings.Split(signedID, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid session ID format")
	}
	sessionID := parts[0]
	return m.store.DeleteSession(ctx, sessionID)
}

// CreateCookie creates an HTTP cookie with a signed session ID
func (m *Manager) CreateCookie(signedSessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    signedSessionID,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure: true, // Enable in production with HTTPS
	}
}

// signSessionID creates an HMAC signature for the session ID
func (m *Manager) signSessionID(sessionID string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(sessionID))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s.%s", sessionID, signature)
}

// validateSignedSessionID verifies the HMAC signature and returns the session ID
func (m *Manager) validateSignedSessionID(signedID string) (string, bool) {
	parts := strings.Split(signedID, ".")
	if len(parts) != 2 {
		return "", false
	}

	sessionID := parts[0]
	providedSignature := parts[1]

	// Compute expected signature
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(sessionID))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Use constant-time comparison to prevent timing attacks
	if !hmac.Equal([]byte(providedSignature), []byte(expectedSignature)) {
		return "", false
	}

	return sessionID, true
}

// generateSessionID creates a cryptographically secure random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate session ID: %v", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// sessionDataFromModel converts model.Session to SessionData
func sessionDataFromModel(s *model.Session) *SessionData {
	return &SessionData{
		SessionType:   SessionType(s.SessionType),
		UserID:        s.UserID,
		CommitteeSlug: s.CommitteeSlug,
		Username:      s.Username,
		Role:          s.Role,
		AttendeeID:    s.AttendeeID,
		MeetingID:     s.MeetingID,
		FullName:      s.FullName,
		IsChair:       s.IsChair,
		ExpiresAt:     s.ExpiresAt,
	}
}
