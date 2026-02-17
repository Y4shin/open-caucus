package session

import (
	"context"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

// Store defines the interface for session storage
// This is implemented by the repository
type Store interface {
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
}
