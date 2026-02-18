package repository

import (
	"context"
	"time"

	"github.com/Y4shin/conference-tool/internal/repository/model"
)

// Repository defines the interface for data persistence and migration management.
type Repository interface {
	// Close closes the underlying database connection.
	Close() error

	// MigrateUp applies all pending migrations.
	MigrateUp() error

	// MigrateDown rolls back all migrations.
	MigrateDown() error

	// MigrateVersion migrates to a specific version.
	// Use version 0 to roll back all migrations.
	MigrateVersion(version uint) error

	// MigrateSteps applies n migration steps.
	// Positive n migrates up, negative n migrates down.
	MigrateSteps(n int) error

	// MigrationVersion returns the current migration version and whether
	// the database is dirty (migration failed midway).
	MigrationVersion() (version uint, dirty bool, err error)

	// User and authentication
	GetUserByCommitteeAndUsername(ctx context.Context, slug, username string) (*model.User, error)
	GetUserByID(ctx context.Context, id int64) (*model.User, error)

	// Committee
	GetCommitteeBySlug(ctx context.Context, slug string) (*model.Committee, error)

	// Session management
	CreateSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, sessionID string) (*model.Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteExpiredSessions(ctx context.Context, before time.Time) error

	// Meetings
	GetMeetingByID(ctx context.Context, id int64) (*model.Meeting, error)
	ListMeetingsForCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.Meeting, error)
	CountMeetingsForCommittee(ctx context.Context, slug string) (int64, error)
	CreateMeeting(ctx context.Context, committeeID int64, name, description, secret string, signupOpen bool) error
	DeleteMeeting(ctx context.Context, id int64) error
	SetActiveMeeting(ctx context.Context, slug string, meetingID int64) error

	// Admin - Committee management
	ListAllCommittees(ctx context.Context, limit, offset int) ([]*model.Committee, error)
	CountAllCommittees(ctx context.Context) (int64, error)
	CreateCommitteeWithSlug(ctx context.Context, name, slug string) error
	DeleteCommitteeBySlug(ctx context.Context, slug string) error
	GetCommitteeIDBySlug(ctx context.Context, slug string) (int64, error)

	// Admin - User management
	ListUsersInCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.User, error)
	CountUsersInCommittee(ctx context.Context, slug string) (int64, error)
	CreateUser(ctx context.Context, committeeID int64, username, passwordHash, fullName string, quoted bool, role string) error
	DeleteUserByID(ctx context.Context, id int64) error
}
