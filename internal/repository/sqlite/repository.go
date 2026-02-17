package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"

	"github.com/Y4shin/conference-tool/internal/repository"
	"github.com/Y4shin/conference-tool/internal/repository/model"
	"github.com/Y4shin/conference-tool/internal/repository/sqlite/client"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Repository wraps the database connection and generated sqlc client.
// It implements the repository.Repository interface.
type Repository struct {
	DB      *sql.DB
	Queries *client.Queries
}

// Ensure Repository implements repository.Repository at compile time.
var _ repository.Repository = (*Repository)(nil)

// New opens a SQLite database at the given path and returns a Repository.
func New(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return &Repository{
		DB:      db,
		Queries: client.New(db),
	}, nil
}

// Close closes the underlying database connection.
func (r *Repository) Close() error {
	return r.DB.Close()
}

func (r *Repository) migrator() (*migrate.Migrate, error) {
	source, err := iofs.New(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create migration source: %w", err)
	}

	driver, err := sqlite.WithInstance(r.DB, &sqlite.Config{})
	if err != nil {
		return nil, fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil
}

// MigrateUp applies all pending migrations.
func (r *Repository) MigrateUp() error {
	m, err := r.migrator()
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

// MigrateDown rolls back all migrations.
func (r *Repository) MigrateDown() error {
	m, err := r.migrator()
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down: %w", err)
	}

	return nil
}

// MigrateVersion migrates to a specific version.
// Use version 0 to roll back all migrations.
func (r *Repository) MigrateVersion(version uint) error {
	m, err := r.migrator()
	if err != nil {
		return err
	}

	if err := m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}

	return nil
}

// MigrateSteps applies n migration steps.
// Positive n migrates up, negative n migrates down.
func (r *Repository) MigrateSteps(n int) error {
	m, err := r.migrator()
	if err != nil {
		return err
	}

	if err := m.Steps(n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate %d steps: %w", n, err)
	}

	return nil
}

// MigrationVersion returns the current migration version and whether
// the database is dirty (migration failed midway).
func (r *Repository) MigrationVersion() (version uint, dirty bool, err error) {
	m, err := r.migrator()
	if err != nil {
		return 0, false, err
	}

	v, d, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, fmt.Errorf("get migration version: %w", err)
	}

	// ErrNilVersion means no migrations have been run yet
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}

	return v, d, nil
}

// GetUserByCommitteeAndUsername retrieves a user by committee slug and username
func (r *Repository) GetUserByCommitteeAndUsername(ctx context.Context, slug, username string) (*model.User, error) {
	user, err := r.Queries.GetUserByCommitteeAndUsername(ctx, client.GetUserByCommitteeAndUsernameParams{
		Slug:     slug,
		Username: username,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user by committee and username: %w", err)
	}
	return userFromClient(&user), nil
}

// GetCommitteeBySlug retrieves a committee by its slug
func (r *Repository) GetCommitteeBySlug(ctx context.Context, slug string) (*model.Committee, error) {
	committee, err := r.Queries.GetCommitteeBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("committee not found")
		}
		return nil, fmt.Errorf("get committee by slug: %w", err)
	}
	return committeeFromClient(&committee), nil
}

// GetUserByID retrieves a user by ID
func (r *Repository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	user, err := r.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return userFromClient(&user), nil
}

// CreateSession stores a new session in the database
func (r *Repository) CreateSession(ctx context.Context, session *model.Session) error {
	var userID, attendeeID, meetingID sql.NullInt64
	var committeeSlug sql.NullString

	if session.UserID != nil {
		userID = sql.NullInt64{Int64: *session.UserID, Valid: true}
	}
	if session.CommitteeSlug != nil {
		committeeSlug = sql.NullString{String: *session.CommitteeSlug, Valid: true}
	}
	if session.AttendeeID != nil {
		attendeeID = sql.NullInt64{Int64: *session.AttendeeID, Valid: true}
	}
	if session.MeetingID != nil {
		meetingID = sql.NullInt64{Int64: *session.MeetingID, Valid: true}
	}

	err := r.Queries.CreateSession(ctx, client.CreateSessionParams{
		SessionID:     session.SessionID,
		SessionType:   string(session.SessionType),
		UserID:        userID,
		CommitteeSlug: committeeSlug,
		AttendeeID:    attendeeID,
		MeetingID:     meetingID,
		ExpiresAt:     session.ExpiresAt.Format("2006-01-02T15:04:05.999Z"),
	})
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID
func (r *Repository) GetSession(ctx context.Context, sessionID string) (*model.Session, error) {
	sess, err := r.Queries.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return sessionFromClient(&sess)
}

// DeleteSession removes a session from the database
func (r *Repository) DeleteSession(ctx context.Context, sessionID string) error {
	err := r.Queries.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all sessions that expired before the given time
func (r *Repository) DeleteExpiredSessions(ctx context.Context, before time.Time) error {
	err := r.Queries.DeleteExpiredSessions(ctx, before.Format("2006-01-02T15:04:05.999Z"))
	if err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

// Converter functions from SQLC client types to model types

func userFromClient(u *client.User) *model.User {
	createdAt, _ := time.Parse(time.RFC3339, u.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, u.UpdatedAt)

	return &model.User{
		ID:           u.ID,
		CommitteeID:  u.CommitteeID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		Quoted:       u.Quoted,
		Role:         u.Role,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func committeeFromClient(c *client.Committee) *model.Committee {
	createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, c.UpdatedAt)

	var currentMeetingID *int64
	if c.CurrentMeetingID.Valid {
		currentMeetingID = &c.CurrentMeetingID.Int64
	}

	return &model.Committee{
		ID:               c.ID,
		Name:             c.Name,
		Slug:             c.Slug,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
		CurrentMeetingID: currentMeetingID,
	}
}

func sessionFromClient(s *client.Session) (*model.Session, error) {
	createdAt, err := time.Parse(time.RFC3339, s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}

	session := &model.Session{
		SessionID:   s.SessionID,
		SessionType: model.SessionType(s.SessionType),
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
	}

	if s.UserID.Valid {
		session.UserID = &s.UserID.Int64
	}
	if s.CommitteeSlug.Valid {
		session.CommitteeSlug = &s.CommitteeSlug.String
	}
	if s.AttendeeID.Valid {
		session.AttendeeID = &s.AttendeeID.Int64
	}
	if s.MeetingID.Valid {
		session.MeetingID = &s.MeetingID.Int64
	}

	// For user sessions, fetch additional user data
	if session.IsUserSession() && session.UserID != nil {
		// Note: This would require a join query in production
		// For now, these fields will be populated by the session manager
	}

	return session, nil
}

// Admin - Committee management

// ListAllCommittees retrieves all committees
func (r *Repository) ListAllCommittees(ctx context.Context) ([]*model.Committee, error) {
	committees, err := r.Queries.ListAllCommittees(ctx)
	if err != nil {
		return nil, fmt.Errorf("list all committees: %w", err)
	}

	result := make([]*model.Committee, len(committees))
	for i := range committees {
		result[i] = committeeFromClient(&committees[i])
	}
	return result, nil
}

// CreateCommitteeWithSlug creates a new committee with a given slug
func (r *Repository) CreateCommitteeWithSlug(ctx context.Context, name, slug string) error {
	_, err := r.Queries.CreateCommitteeWithSlug(ctx, client.CreateCommitteeWithSlugParams{
		Name: name,
		Slug: slug,
	})
	if err != nil {
		return fmt.Errorf("create committee: %w", err)
	}
	return nil
}

// DeleteCommitteeBySlug deletes a committee by slug
func (r *Repository) DeleteCommitteeBySlug(ctx context.Context, slug string) error {
	err := r.Queries.DeleteCommitteeBySlug(ctx, slug)
	if err != nil {
		return fmt.Errorf("delete committee: %w", err)
	}
	return nil
}

// GetCommitteeIDBySlug retrieves a committee ID by slug
func (r *Repository) GetCommitteeIDBySlug(ctx context.Context, slug string) (int64, error) {
	id, err := r.Queries.GetCommitteeIDBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("committee not found")
		}
		return 0, fmt.Errorf("get committee ID: %w", err)
	}
	return id, nil
}

// Admin - User management

// ListUsersInCommittee retrieves all users in a committee
func (r *Repository) ListUsersInCommittee(ctx context.Context, slug string) ([]*model.User, error) {
	users, err := r.Queries.ListUsersInCommittee(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("list users in committee: %w", err)
	}

	result := make([]*model.User, len(users))
	for i := range users {
		result[i] = userFromClient(&users[i])
	}
	return result, nil
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, committeeID int64, username, passwordHash, fullName string, quoted bool, role string) error {
	_, err := r.Queries.CreateUser(ctx, client.CreateUserParams{
		CommitteeID:  committeeID,
		Username:     username,
		PasswordHash: passwordHash,
		FullName:     fullName,
		Quoted:       quoted,
		Role:         role,
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// DeleteUserByID deletes a user by ID
func (r *Repository) DeleteUserByID(ctx context.Context, id int64) error {
	err := r.Queries.DeleteUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
