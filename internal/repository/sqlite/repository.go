package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
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

	// SQLite supports only one writer at a time. Limiting to one open connection
	// also ensures :memory: databases share a single connection across all callers.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

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

// GetAccountByUsername retrieves a sitewide account by username
func (r *Repository) GetAccountByUsername(ctx context.Context, username string) (*model.Account, error) {
	account, err := r.Queries.GetAccountByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("get account by username: %w", err)
	}
	return accountFromClient(&account), nil
}

// GetAccountByID retrieves a sitewide account by ID
func (r *Repository) GetAccountByID(ctx context.Context, id int64) (*model.Account, error) {
	account, err := r.Queries.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("get account by id: %w", err)
	}
	return accountFromClient(&account), nil
}

// GetPasswordCredential retrieves the password credential for an account.
func (r *Repository) GetPasswordCredential(ctx context.Context, accountID int64) (*model.PasswordCredential, error) {
	cred, err := r.Queries.GetPasswordCredentialByAccountID(ctx, accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("password credential not found: %w", err)
		}
		return nil, fmt.Errorf("get password credential: %w", err)
	}
	return passwordCredentialFromClient(&cred), nil
}

// GetUserByCommitteeAndUsername retrieves a committee membership by committee slug and username
func (r *Repository) GetUserByCommitteeAndUsername(ctx context.Context, slug, username string) (*model.User, error) {
	row, err := r.Queries.GetUserMembershipByAccountAndCommittee(ctx, client.GetUserMembershipByAccountAndCommitteeParams{
		Username: username,
		Slug:     slug,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user by committee and username: %w", err)
	}
	return userFromMembershipRow(&row), nil
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

// GetUserByID retrieves a membership row by ID, including the username from accounts
func (r *Repository) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	row, err := r.Queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return userFromGetUserByIDRow(&row), nil
}

// GetUserMembershipByAccountIDAndSlug retrieves a committee membership by account ID and committee slug
func (r *Repository) GetUserMembershipByAccountIDAndSlug(ctx context.Context, accountID int64, slug string) (*model.User, error) {
	row, err := r.Queries.GetUserMembershipByAccountIDAndSlug(ctx, client.GetUserMembershipByAccountIDAndSlugParams{
		AccountID: accountID,
		Slug:      slug,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("get user membership by account id and slug: %w", err)
	}
	return userFromMembershipByAccountIDAndSlugRow(&row), nil
}

// ListCommitteesByAccountID returns all committees an account has a membership in
func (r *Repository) ListCommitteesByAccountID(ctx context.Context, accountID int64) ([]*model.Committee, error) {
	rows, err := r.Queries.ListCommitteesByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("list committees by account id: %w", err)
	}
	result := make([]*model.Committee, len(rows))
	for i := range rows {
		result[i] = committeeFromListByAccountIDRow(&rows[i])
	}
	return result, nil
}

// CreateSession stores a new session in the database
func (r *Repository) CreateSession(ctx context.Context, session *model.Session) error {
	var accountID, attendeeID sql.NullInt64

	if session.AccountID != nil {
		accountID = sql.NullInt64{Int64: *session.AccountID, Valid: true}
	}
	if session.AttendeeID != nil {
		attendeeID = sql.NullInt64{Int64: *session.AttendeeID, Valid: true}
	}

	err := r.Queries.CreateSession(ctx, client.CreateSessionParams{
		SessionID:   session.SessionID,
		SessionType: string(session.SessionType),
		AccountID:   accountID,
		AttendeeID:  attendeeID,
		ExpiresAt:   session.ExpiresAt.Format("2006-01-02T15:04:05.999Z"),
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

func accountFromClient(a *client.Account) *model.Account {
	createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, a.UpdatedAt)

	return &model.Account{
		ID:         a.ID,
		Username:   a.Username,
		AuthMethod: a.AuthMethod,
		IsAdmin:    a.IsAdmin,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
}

func passwordCredentialFromClient(c *client.PasswordCredential) *model.PasswordCredential {
	createdAt, _ := time.Parse(time.RFC3339, c.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, c.UpdatedAt)

	return &model.PasswordCredential{
		ID:           c.ID,
		AccountID:    c.AccountID,
		AuthMethod:   c.AuthMethod,
		PasswordHash: c.PasswordHash,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}

func userFromGetUserByIDRow(r *client.GetUserByIDRow) *model.User {
	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &model.User{
		ID:          r.ID,
		AccountID:   r.AccountID,
		CommitteeID: r.CommitteeID,
		Username:    r.Username,
		FullName:    r.FullName,
		Quoted:      r.Quoted,
		Role:        r.Role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func userFromMembershipRow(r *client.GetUserMembershipByAccountAndCommitteeRow) *model.User {
	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &model.User{
		ID:          r.ID,
		AccountID:   r.AccountID,
		CommitteeID: r.CommitteeID,
		Username:    r.Username,
		FullName:    r.FullName,
		Quoted:      r.Quoted,
		Role:        r.Role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func userFromMembershipByAccountIDAndSlugRow(r *client.GetUserMembershipByAccountIDAndSlugRow) *model.User {
	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &model.User{
		ID:            r.ID,
		AccountID:     r.AccountID,
		CommitteeID:   r.CommitteeID,
		Username:      r.Username,
		CommitteeSlug: r.CommitteeSlug,
		FullName:      r.FullName,
		Quoted:        r.Quoted,
		Role:          r.Role,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func committeeFromListByAccountIDRow(r *client.ListCommitteesByAccountIDRow) *model.Committee {
	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &model.Committee{
		ID:        r.ID,
		Name:      r.Name,
		Slug:      r.Slug,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func userFromListRow(r *client.ListUsersInCommitteeRow) *model.User {
	createdAt, _ := time.Parse(time.RFC3339, r.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, r.UpdatedAt)

	return &model.User{
		ID:          r.ID,
		AccountID:   r.AccountID,
		CommitteeID: r.CommitteeID,
		Username:    r.Username,
		FullName:    r.FullName,
		Quoted:      r.Quoted,
		Role:        r.Role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
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

	if s.AccountID.Valid {
		session.AccountID = &s.AccountID.Int64
	}
	if s.AttendeeID.Valid {
		session.AttendeeID = &s.AttendeeID.Int64
	}

	return session, nil
}

// Admin - Committee management

// ListAllCommittees retrieves a page of committees
func (r *Repository) ListAllCommittees(ctx context.Context, limit, offset int) ([]*model.Committee, error) {
	committees, err := r.Queries.ListAllCommittees(ctx, client.ListAllCommitteesParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list all committees: %w", err)
	}

	result := make([]*model.Committee, len(committees))
	for i := range committees {
		result[i] = committeeFromClient(&committees[i])
	}
	return result, nil
}

// CountAllCommittees returns the total number of committees
func (r *Repository) CountAllCommittees(ctx context.Context) (int64, error) {
	count, err := r.Queries.CountAllCommittees(ctx)
	if err != nil {
		return 0, fmt.Errorf("count all committees: %w", err)
	}
	return count, nil
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

// ListUsersInCommittee retrieves a page of users in a committee
func (r *Repository) ListUsersInCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.User, error) {
	users, err := r.Queries.ListUsersInCommittee(ctx, client.ListUsersInCommitteeParams{
		Slug:   slug,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list users in committee: %w", err)
	}

	result := make([]*model.User, len(users))
	for i := range users {
		result[i] = userFromListRow(&users[i])
	}
	return result, nil
}

// CountUsersInCommittee returns the total number of users in a committee
func (r *Repository) CountUsersInCommittee(ctx context.Context, slug string) (int64, error) {
	count, err := r.Queries.CountUsersInCommittee(ctx, slug)
	if err != nil {
		return 0, fmt.Errorf("count users in committee: %w", err)
	}
	return count, nil
}

// CreateUser creates a committee membership for a sitewide account.
// If no account with the given username exists, one is created using the provided
// passwordHash. If an account already exists, it is reused and the passwordHash
// is ignored.
func (r *Repository) CreateUser(ctx context.Context, committeeID int64, username, passwordHash, fullName string, quoted bool, role string) error {
	account, err := r.Queries.GetAccountByUsername(ctx, username)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("create user: look up account: %w", err)
		}
		// Account does not exist yet — create it.
		account, err = r.Queries.CreateAccount(ctx, username)
		if err != nil {
			return fmt.Errorf("create user: create account: %w", err)
		}
		if _, err := r.Queries.CreatePasswordCredential(ctx, client.CreatePasswordCredentialParams{
			AccountID:    account.ID,
			PasswordHash: passwordHash,
		}); err != nil {
			return fmt.Errorf("create user: create account credential: %w", err)
		}
	}

	_, err = r.Queries.CreateMembership(ctx, client.CreateMembershipParams{
		AccountID:   account.ID,
		CommitteeID: committeeID,
		FullName:    fullName,
		Role:        role,
		Quoted:      quoted,
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

// SetAccountIsAdmin sets or clears the is_admin flag on an account
func (r *Repository) SetAccountIsAdmin(ctx context.Context, accountID int64, isAdmin bool) error {
	err := r.Queries.SetAccountIsAdmin(ctx, client.SetAccountIsAdminParams{
		IsAdmin: isAdmin,
		ID:      accountID,
	})
	if err != nil {
		return fmt.Errorf("set account is_admin: %w", err)
	}
	return nil
}

// CreateAccount creates a new sitewide account with the given username and password hash.
func (r *Repository) CreateAccount(ctx context.Context, username, passwordHash string) (*model.Account, error) {
	account, err := r.Queries.CreateAccount(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	if _, err := r.Queries.CreatePasswordCredential(ctx, client.CreatePasswordCredentialParams{
		AccountID:    account.ID,
		PasswordHash: passwordHash,
	}); err != nil {
		return nil, fmt.Errorf("create account credential: %w", err)
	}
	return accountFromClient(&account), nil
}

// ListMeetingsForCommittee retrieves a page of meetings for a committee by slug
func (r *Repository) ListMeetingsForCommittee(ctx context.Context, slug string, limit, offset int) ([]*model.Meeting, error) {
	meetings, err := r.Queries.ListMeetingsForCommittee(ctx, client.ListMeetingsForCommitteeParams{
		Slug:   slug,
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list meetings: %w", err)
	}
	result := make([]*model.Meeting, len(meetings))
	for i := range meetings {
		result[i] = meetingFromClient(&meetings[i])
	}
	return result, nil
}

// CountMeetingsForCommittee returns the total number of meetings for a committee
func (r *Repository) CountMeetingsForCommittee(ctx context.Context, slug string) (int64, error) {
	count, err := r.Queries.CountMeetingsForCommittee(ctx, slug)
	if err != nil {
		return 0, fmt.Errorf("count meetings: %w", err)
	}
	return count, nil
}

// CreateMeeting creates a new meeting for a committee
func (r *Repository) CreateMeeting(ctx context.Context, committeeID int64, name, description, secret string, signupOpen bool) error {
	err := r.Queries.CreateMeeting(ctx, client.CreateMeetingParams{
		CommitteeID: committeeID,
		Name:        name,
		Description: description,
		Secret:      secret,
		SignupOpen:  signupOpen,
	})
	if err != nil {
		return fmt.Errorf("create meeting: %w", err)
	}
	return nil
}

// GetMeetingByID retrieves a meeting by its ID
func (r *Repository) GetMeetingByID(ctx context.Context, id int64) (*model.Meeting, error) {
	m, err := r.Queries.GetMeetingByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("meeting not found")
		}
		return nil, fmt.Errorf("get meeting: %w", err)
	}
	return meetingFromClient(&m), nil
}

// DeleteMeeting removes a meeting by ID
func (r *Repository) DeleteMeeting(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteMeeting(ctx, id); err != nil {
		return fmt.Errorf("delete meeting: %w", err)
	}
	return nil
}

// SetActiveMeeting sets a meeting as the current active meeting for a committee
func (r *Repository) SetActiveMeeting(ctx context.Context, slug string, meetingID int64) error {
	err := r.Queries.SetActiveMeeting(ctx, client.SetActiveMeetingParams{
		CurrentMeetingID: sql.NullInt64{Int64: meetingID, Valid: true},
		Slug:             slug,
	})
	if err != nil {
		return fmt.Errorf("set active meeting: %w", err)
	}
	return nil
}

// SetMeetingSignupOpen updates the signup_open flag for a meeting
func (r *Repository) SetMeetingSignupOpen(ctx context.Context, id int64, open bool) error {
	err := r.Queries.SetMeetingSignupOpen(ctx, client.SetMeetingSignupOpenParams{
		SignupOpen: open,
		ID:         id,
	})
	if err != nil {
		return fmt.Errorf("set meeting signup_open: %w", err)
	}
	return nil
}

// CreateAttendee creates a new attendee row for a meeting
func (r *Repository) CreateAttendee(ctx context.Context, meetingID int64, userID *int64, fullName, secret string, quoted bool) (*model.Attendee, error) {
	var uid sql.NullInt64
	if userID != nil {
		uid = sql.NullInt64{Int64: *userID, Valid: true}
	}
	a, err := r.Queries.CreateAttendee(ctx, client.CreateAttendeeParams{
		MeetingID:   meetingID,
		UserID:      uid,
		FullName:    fullName,
		Secret:      secret,
		Quoted:      quoted,
		MeetingID_2: meetingID,
	})
	if err != nil {
		return nil, fmt.Errorf("create attendee: %w", err)
	}
	return attendeeFromClient(&a), nil
}

// GetAttendeeByUserIDAndMeetingID retrieves an attendee by their user ID and meeting ID
func (r *Repository) GetAttendeeByUserIDAndMeetingID(ctx context.Context, userID, meetingID int64) (*model.Attendee, error) {
	a, err := r.Queries.GetAttendeeByUserIDAndMeetingID(ctx, client.GetAttendeeByUserIDAndMeetingIDParams{
		UserID:    sql.NullInt64{Int64: userID, Valid: true},
		MeetingID: meetingID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("attendee not found")
		}
		return nil, fmt.Errorf("get attendee: %w", err)
	}
	return attendeeFromClient(&a), nil
}

// GetAttendeeByID retrieves an attendee by their ID
func (r *Repository) GetAttendeeByID(ctx context.Context, id int64) (*model.Attendee, error) {
	a, err := r.Queries.GetAttendeeByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("attendee not found")
		}
		return nil, fmt.Errorf("get attendee: %w", err)
	}
	return attendeeFromClient(&a), nil
}

// GetAttendeeByMeetingIDAndSecret retrieves an attendee by meeting ID and their secret (for login)
func (r *Repository) GetAttendeeByMeetingIDAndSecret(ctx context.Context, meetingID int64, secret string) (*model.Attendee, error) {
	a, err := r.Queries.GetAttendeeByMeetingIDAndSecret(ctx, client.GetAttendeeByMeetingIDAndSecretParams{
		MeetingID: meetingID,
		Secret:    secret,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("attendee not found")
		}
		return nil, fmt.Errorf("get attendee by secret: %w", err)
	}
	return attendeeFromClient(&a), nil
}

// ListAttendeesForMeeting returns all attendees for a meeting ordered by creation time
func (r *Repository) ListAttendeesForMeeting(ctx context.Context, meetingID int64) ([]*model.Attendee, error) {
	rows, err := r.Queries.ListAttendeesForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("list attendees: %w", err)
	}
	result := make([]*model.Attendee, len(rows))
	for i := range rows {
		result[i] = attendeeFromClient(&rows[i])
	}
	return result, nil
}

// DeleteAttendee removes an attendee by ID
func (r *Repository) DeleteAttendee(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteAttendee(ctx, id); err != nil {
		return fmt.Errorf("delete attendee: %w", err)
	}
	return nil
}

// SetAttendeeIsChair updates the is_chair flag for an attendee
func (r *Repository) SetAttendeeIsChair(ctx context.Context, id int64, isChair bool) error {
	err := r.Queries.SetAttendeeIsChair(ctx, client.SetAttendeeIsChairParams{
		IsChair: isChair,
		ID:      id,
	})
	if err != nil {
		return fmt.Errorf("set attendee is_chair: %w", err)
	}
	return nil
}

// SetAttendeeQuoted updates the quoted flag for an attendee.
func (r *Repository) SetAttendeeQuoted(ctx context.Context, id int64, quoted bool) error {
	err := r.Queries.SetAttendeeQuoted(ctx, client.SetAttendeeQuotedParams{
		Quoted: quoted,
		ID:     id,
	})
	if err != nil {
		return fmt.Errorf("set attendee quoted: %w", err)
	}
	return nil
}

func attendeeFromClient(a *client.Attendee) *model.Attendee {
	createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
	var userID *int64
	if a.UserID.Valid {
		userID = &a.UserID.Int64
	}
	var attendeeNumber int64
	if a.AttendeeNumber.Valid {
		attendeeNumber = a.AttendeeNumber.Int64
	}
	return &model.Attendee{
		ID:             a.ID,
		MeetingID:      a.MeetingID,
		AttendeeNumber: attendeeNumber,
		UserID:         userID,
		FullName:       a.FullName,
		Secret:         a.Secret,
		IsChair:        a.IsChair,
		Quoted:         a.Quoted,
		CreatedAt:      createdAt,
	}
}

func meetingFromClient(m *client.Meeting) *model.Meeting {
	createdAt, _ := time.Parse(time.RFC3339, m.CreatedAt)
	var currentAgendaPointID *int64
	if m.CurrentAgendaPointID.Valid {
		currentAgendaPointID = &m.CurrentAgendaPointID.Int64
	}
	var protocolWriterID *int64
	if m.ProtocolWriterID.Valid {
		protocolWriterID = &m.ProtocolWriterID.Int64
	}
	return &model.Meeting{
		ID:                           m.ID,
		Name:                         m.Name,
		Description:                  m.Description,
		Secret:                       m.Secret,
		SignupOpen:                   m.SignupOpen,
		CurrentAgendaPointID:         currentAgendaPointID,
		ProtocolWriterID:             protocolWriterID,
		GenderQuotationEnabled:       m.GenderQuotationEnabled,
		FirstSpeakerQuotationEnabled: m.FirstSpeakerQuotationEnabled,
		ModeratorID:                  nullInt64ToPtr(m.ModeratorID),
		CreatedAt:                    createdAt,
	}
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	return &n.Int64
}

func nullBoolToPtr(n sql.NullBool) *bool {
	if !n.Valid {
		return nil
	}
	return &n.Bool
}

func ptrToNullBool(p *bool) sql.NullBool {
	if p == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *p, Valid: true}
}

func agendaPointFromClient(ap *client.AgendaPoint) *model.AgendaPoint {
	var currentSpeakerID *int64
	if ap.CurrentSpeakerID.Valid {
		currentSpeakerID = &ap.CurrentSpeakerID.Int64
	}
	var parentID *int64
	if ap.ParentID.Valid {
		parentID = &ap.ParentID.Int64
	}
	return &model.AgendaPoint{
		ID:                           ap.ID,
		MeetingID:                    ap.MeetingID,
		ParentID:                     parentID,
		Position:                     ap.Position,
		Title:                        ap.Title,
		Protocol:                     ap.Protocol,
		CurrentSpeakerID:             currentSpeakerID,
		GenderQuotationEnabled:       nullBoolToPtr(ap.GenderQuotationEnabled),
		FirstSpeakerQuotationEnabled: nullBoolToPtr(ap.FirstSpeakerQuotationEnabled),
		ModeratorID:                  nullInt64ToPtr(ap.ModeratorID),
	}
}

// CreateAgendaPoint creates a new top-level agenda point for a meeting.
func (r *Repository) CreateAgendaPoint(ctx context.Context, meetingID int64, title string) (*model.AgendaPoint, error) {
	posRaw, err := r.Queries.GetMaxAgendaPointPosition(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("get max position: %w", err)
	}
	var pos int64
	switch v := posRaw.(type) {
	case int64:
		pos = v + 1
	case float64:
		pos = int64(v) + 1
	default:
		pos = 1
	}
	ap, err := r.Queries.CreateAgendaPoint(ctx, client.CreateAgendaPointParams{
		MeetingID: meetingID,
		Position:  pos,
		Title:     title,
	})
	if err != nil {
		return nil, fmt.Errorf("create agenda point: %w", err)
	}
	return agendaPointFromClient(&ap), nil
}

// CreateSubAgendaPoint creates a new child agenda point for a parent agenda point.
func (r *Repository) CreateSubAgendaPoint(ctx context.Context, meetingID, parentID int64, title string) (*model.AgendaPoint, error) {
	posRaw, err := r.Queries.GetMaxSubAgendaPointPosition(ctx, client.GetMaxSubAgendaPointPositionParams{
		MeetingID: meetingID,
		ParentID:  sql.NullInt64{Int64: parentID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("get max sub-agenda position: %w", err)
	}
	var pos int64
	switch v := posRaw.(type) {
	case int64:
		pos = v + 1
	case float64:
		pos = int64(v) + 1
	default:
		pos = 1
	}
	ap, err := r.Queries.CreateSubAgendaPoint(ctx, client.CreateSubAgendaPointParams{
		MeetingID: meetingID,
		ParentID:  sql.NullInt64{Int64: parentID, Valid: true},
		Position:  pos,
		Title:     title,
	})
	if err != nil {
		return nil, fmt.Errorf("create sub-agenda point: %w", err)
	}
	return agendaPointFromClient(&ap), nil
}

// ListAgendaPointsForMeeting returns all top-level agenda points for a meeting.
func (r *Repository) ListAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error) {
	rows, err := r.Queries.ListAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("list agenda points: %w", err)
	}
	result := make([]*model.AgendaPoint, len(rows))
	for i := range rows {
		result[i] = agendaPointFromClient(&rows[i])
	}
	return result, nil
}

// ListSubAgendaPointsForMeeting returns all child agenda points for a meeting.
func (r *Repository) ListSubAgendaPointsForMeeting(ctx context.Context, meetingID int64) ([]*model.AgendaPoint, error) {
	rows, err := r.Queries.ListSubAgendaPointsForMeeting(ctx, meetingID)
	if err != nil {
		return nil, fmt.Errorf("list sub-agenda points: %w", err)
	}
	result := make([]*model.AgendaPoint, len(rows))
	for i := range rows {
		result[i] = agendaPointFromClient(&rows[i])
	}
	return result, nil
}

// GetAgendaPointByID retrieves an agenda point by ID.
func (r *Repository) GetAgendaPointByID(ctx context.Context, id int64) (*model.AgendaPoint, error) {
	ap, err := r.Queries.GetAgendaPointByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("agenda point not found")
		}
		return nil, fmt.Errorf("get agenda point: %w", err)
	}
	return agendaPointFromClient(&ap), nil
}

// DeleteAgendaPoint removes an agenda point by ID.
func (r *Repository) DeleteAgendaPoint(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteAgendaPoint(ctx, id); err != nil {
		return fmt.Errorf("delete agenda point: %w", err)
	}
	return nil
}

// SetCurrentAgendaPoint sets the active agenda point for a meeting (nil clears it).
func (r *Repository) SetCurrentAgendaPoint(ctx context.Context, meetingID int64, agendaPointID *int64) error {
	var apID sql.NullInt64
	if agendaPointID != nil {
		apID = sql.NullInt64{Int64: *agendaPointID, Valid: true}
	}
	if err := r.Queries.SetCurrentAgendaPoint(ctx, client.SetCurrentAgendaPointParams{
		CurrentAgendaPointID: apID,
		ID:                   meetingID,
	}); err != nil {
		return fmt.Errorf("set current agenda point: %w", err)
	}
	return nil
}

// AddSpeaker adds an attendee to the speakers list for an agenda point.
func (r *Repository) AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error) {
	// RoPM entries never receive the first-speaker flag.
	if speakerType == "ropm" {
		firstSpeaker = false
	}

	row, err := r.Queries.AddSpeaker(ctx, client.AddSpeakerParams{
		AgendaPointID: agendaPointID,
		AttendeeID:    attendeeID,
		Type:          speakerType,
		GenderQuoted:  genderQuoted,
		FirstSpeaker:  firstSpeaker,
	})
	if err != nil {
		return nil, fmt.Errorf("add speaker: %w", err)
	}
	return speakerFromRow(row.ID, row.AgendaPointID, row.AttendeeID, "", row.Type, row.Status, row.StartOfSpeech, row.Duration, row.GenderQuoted, row.FirstSpeaker, row.Priority, row.OrderPosition), nil
}

// speakerFromRow constructs a model.SpeakerEntry from individual column values.
func parseStartOfSpeech(startOfSpeech sql.NullString) *time.Time {
	if !startOfSpeech.Valid || startOfSpeech.String == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, startOfSpeech.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseDurationSeconds(duration sql.NullString) int64 {
	if !duration.Valid || duration.String == "" {
		return 0
	}

	s := strings.TrimSpace(duration.String)
	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		if secs < 0 {
			return 0
		}
		return secs
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	secs := int64(parsed / time.Second)
	if secs < 0 {
		return 0
	}
	return secs
}

func speakerFromRow(
	id, agendaPointID, attendeeID int64,
	attendeeName, typ, status string,
	startOfSpeech sql.NullString,
	duration sql.NullString,
	genderQuoted, firstSpeaker, priority bool,
	orderPosition int64,
) *model.SpeakerEntry {
	return &model.SpeakerEntry{
		ID:              id,
		AgendaPointID:   agendaPointID,
		AttendeeID:      attendeeID,
		AttendeeName:    attendeeName,
		Type:            typ,
		Status:          status,
		GenderQuoted:    genderQuoted,
		FirstSpeaker:    firstSpeaker,
		Priority:        priority,
		OrderPosition:   orderPosition,
		StartOfSpeech:   parseStartOfSpeech(startOfSpeech),
		DurationSeconds: parseDurationSeconds(duration),
	}
}

// ListSpeakersForAgendaPoint returns all speakers for an agenda point.
func (r *Repository) ListSpeakersForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.SpeakerEntry, error) {
	rows, err := r.Queries.ListSpeakersForAgendaPoint(ctx, agendaPointID)
	if err != nil {
		return nil, fmt.Errorf("list speakers: %w", err)
	}
	result := make([]*model.SpeakerEntry, len(rows))
	for i, row := range rows {
		result[i] = speakerFromRow(row.ID, row.AgendaPointID, row.AttendeeID, row.AttendeeFullName, row.Type, row.Status, row.StartOfSpeech, row.Duration, row.GenderQuoted, row.FirstSpeaker, row.Priority, row.OrderPosition)
	}
	return result, nil
}

// GetSpeakerEntryByID retrieves a speaker entry by ID.
func (r *Repository) GetSpeakerEntryByID(ctx context.Context, id int64) (*model.SpeakerEntry, error) {
	row, err := r.Queries.GetSpeakerEntryByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("speaker entry not found")
		}
		return nil, fmt.Errorf("get speaker entry: %w", err)
	}
	return speakerFromRow(row.ID, row.AgendaPointID, row.AttendeeID, "", row.Type, row.Status, row.StartOfSpeech, row.Duration, row.GenderQuoted, row.FirstSpeaker, row.Priority, row.OrderPosition), nil
}

// DeleteSpeaker removes a speaker entry.
func (r *Repository) DeleteSpeaker(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteSpeaker(ctx, id); err != nil {
		return fmt.Errorf("delete speaker: %w", err)
	}
	return nil
}

// SetSpeakerSpeaking transitions a speaker entry to SPEAKING and records start time.
// It also sets the agenda point's current_speaker_id.
func (r *Repository) SetSpeakerSpeaking(ctx context.Context, id, agendaPointID int64) error {
	if err := r.Queries.SetSpeakerSpeaking(ctx, id); err != nil {
		return fmt.Errorf("set speaker speaking: %w", err)
	}
	if err := r.Queries.SetCurrentSpeaker(ctx, client.SetCurrentSpeakerParams{
		CurrentSpeakerID: sql.NullInt64{Int64: id, Valid: true},
		ID:               agendaPointID,
	}); err != nil {
		return fmt.Errorf("set current speaker: %w", err)
	}
	return nil
}

// SetSpeakerDone transitions a speaker entry to DONE and clears the agenda point's current speaker.
func (r *Repository) SetSpeakerDone(ctx context.Context, id int64) error {
	// Look up the entry to find the agenda_point_id for clearing current_speaker_id.
	entry, err := r.Queries.GetSpeakerEntryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get speaker entry for done: %w", err)
	}
	elapsedSeconds := int64(0)
	if entry.StartOfSpeech.Valid && entry.StartOfSpeech.String != "" {
		if start, parseErr := time.Parse(time.RFC3339Nano, entry.StartOfSpeech.String); parseErr == nil {
			elapsedSeconds = int64(time.Since(start) / time.Second)
			if elapsedSeconds < 0 {
				elapsedSeconds = 0
			}
		}
	}
	if err := r.Queries.SetSpeakerDone(ctx, client.SetSpeakerDoneParams{
		Duration: sql.NullString{String: fmt.Sprintf("%ds", elapsedSeconds), Valid: true},
		ID:       id,
	}); err != nil {
		return fmt.Errorf("set speaker done: %w", err)
	}
	if err := r.Queries.SetCurrentSpeaker(ctx, client.SetCurrentSpeakerParams{
		CurrentSpeakerID: sql.NullInt64{},
		ID:               entry.AgendaPointID,
	}); err != nil {
		return fmt.Errorf("clear current speaker: %w", err)
	}
	return nil
}

// SetSpeakerWithdrawn removes a speaker entry (legacy name kept for compatibility).
func (r *Repository) SetSpeakerWithdrawn(ctx context.Context, id int64) error {
	if err := r.Queries.SetSpeakerWithdrawn(ctx, id); err != nil {
		return fmt.Errorf("remove speaker: %w", err)
	}
	return nil
}

// HasAttendeeSpokenOnAgendaPoint returns true if the attendee has any regular SPEAKING or DONE entry for the agenda point.
func (r *Repository) HasAttendeeSpokenOnAgendaPoint(ctx context.Context, agendaPointID, attendeeID int64) (bool, error) {
	v, err := r.Queries.HasAttendeeSpokenOnAgendaPoint(ctx, client.HasAttendeeSpokenOnAgendaPointParams{
		AgendaPointID: agendaPointID,
		AttendeeID:    attendeeID,
	})
	if err != nil {
		return false, fmt.Errorf("has attendee spoken: %w", err)
	}
	return v != 0, nil
}

// SetSpeakerPriority toggles the manual priority flag on a speakers-list entry.
func (r *Repository) SetSpeakerPriority(ctx context.Context, id int64, priority bool) error {
	if err := r.Queries.SetSpeakerPriority(ctx, client.SetSpeakerPriorityParams{
		ID:       id,
		Priority: priority,
	}); err != nil {
		return fmt.Errorf("set speaker priority: %w", err)
	}
	return nil
}

// RecomputeSpeakerOrder recomputes and persists order_position for all WAITING speakers on an agenda point.
// Sort key: ropm first, then priority DESC, gender_quoted DESC, first_speaker DESC, requested_at ASC.
func (r *Repository) RecomputeSpeakerOrder(ctx context.Context, agendaPointID int64) error {
	rows, err := r.Queries.GetWaitingSpeakersForAgendaPoint(ctx, agendaPointID)
	if err != nil {
		return fmt.Errorf("recompute speaker order fetch: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		aRopm, bRopm := a.Type == "ropm", b.Type == "ropm"
		if aRopm != bRopm {
			return aRopm
		}
		if a.Priority != b.Priority {
			return a.Priority
		}
		if a.GenderQuoted != b.GenderQuoted {
			return a.GenderQuoted
		}
		if a.FirstSpeaker != b.FirstSpeaker {
			return a.FirstSpeaker
		}
		return a.RequestedAt < b.RequestedAt
	})

	for i, row := range rows {
		if err := r.Queries.SetSpeakerOrderPosition(ctx, client.SetSpeakerOrderPositionParams{
			ID:            row.ID,
			OrderPosition: int64(i + 1),
		}); err != nil {
			return fmt.Errorf("recompute speaker order set position: %w", err)
		}
	}
	return nil
}

// SetMeetingGenderQuotation updates the gender_quotation_enabled flag for a meeting.
func (r *Repository) SetMeetingGenderQuotation(ctx context.Context, id int64, enabled bool) error {
	if err := r.Queries.SetMeetingGenderQuotation(ctx, client.SetMeetingGenderQuotationParams{
		ID:                     id,
		GenderQuotationEnabled: enabled,
	}); err != nil {
		return fmt.Errorf("set meeting gender quotation: %w", err)
	}
	return nil
}

// SetMeetingFirstSpeakerQuotation updates the first_speaker_quotation_enabled flag for a meeting.
func (r *Repository) SetMeetingFirstSpeakerQuotation(ctx context.Context, id int64, enabled bool) error {
	if err := r.Queries.SetMeetingFirstSpeakerQuotation(ctx, client.SetMeetingFirstSpeakerQuotationParams{
		ID:                           id,
		FirstSpeakerQuotationEnabled: enabled,
	}); err != nil {
		return fmt.Errorf("set meeting first speaker quotation: %w", err)
	}
	return nil
}

// SetMeetingModerator sets or clears the moderator_id for a meeting.
func (r *Repository) SetMeetingModerator(ctx context.Context, id int64, moderatorID *int64) error {
	var mid sql.NullInt64
	if moderatorID != nil {
		mid = sql.NullInt64{Int64: *moderatorID, Valid: true}
	}
	if err := r.Queries.SetMeetingModerator(ctx, client.SetMeetingModeratorParams{
		ID:          id,
		ModeratorID: mid,
	}); err != nil {
		return fmt.Errorf("set meeting moderator: %w", err)
	}
	return nil
}

// SetAgendaPointGenderQuotation sets the gender_quotation_enabled override for an agenda point (nil clears it).
func (r *Repository) SetAgendaPointGenderQuotation(ctx context.Context, id int64, enabled *bool) error {
	if err := r.Queries.SetAgendaPointGenderQuotation(ctx, client.SetAgendaPointGenderQuotationParams{
		ID:                     id,
		GenderQuotationEnabled: ptrToNullBool(enabled),
	}); err != nil {
		return fmt.Errorf("set agenda point gender quotation: %w", err)
	}
	return nil
}

// SetAgendaPointFirstSpeakerQuotation sets the first_speaker_quotation_enabled override for an agenda point (nil clears it).
func (r *Repository) SetAgendaPointFirstSpeakerQuotation(ctx context.Context, id int64, enabled *bool) error {
	if err := r.Queries.SetAgendaPointFirstSpeakerQuotation(ctx, client.SetAgendaPointFirstSpeakerQuotationParams{
		ID:                           id,
		FirstSpeakerQuotationEnabled: ptrToNullBool(enabled),
	}); err != nil {
		return fmt.Errorf("set agenda point first speaker quotation: %w", err)
	}
	return nil
}

// SetAgendaPointModerator sets or clears the moderator_id for an agenda point.
func (r *Repository) SetAgendaPointModerator(ctx context.Context, id int64, moderatorID *int64) error {
	var mid sql.NullInt64
	if moderatorID != nil {
		mid = sql.NullInt64{Int64: *moderatorID, Valid: true}
	}
	if err := r.Queries.SetAgendaPointModerator(ctx, client.SetAgendaPointModeratorParams{
		ID:          id,
		ModeratorID: mid,
	}); err != nil {
		return fmt.Errorf("set agenda point moderator: %w", err)
	}
	return nil
}

// SetProtocolWriter sets or clears the protocol_writer_id for a meeting.
func (r *Repository) SetProtocolWriter(ctx context.Context, meetingID int64, attendeeID *int64) error {
	var aid sql.NullInt64
	if attendeeID != nil {
		aid = sql.NullInt64{Int64: *attendeeID, Valid: true}
	}
	if err := r.Queries.SetMeetingProtocolWriter(ctx, client.SetMeetingProtocolWriterParams{
		ProtocolWriterID: aid,
		ID:               meetingID,
	}); err != nil {
		return fmt.Errorf("set protocol writer: %w", err)
	}
	return nil
}

// UpdateAgendaPointProtocol saves the protocol text for an agenda point.
func (r *Repository) UpdateAgendaPointProtocol(ctx context.Context, agendaPointID int64, protocol string) error {
	if err := r.Queries.UpdateAgendaPointProtocol(ctx, client.UpdateAgendaPointProtocolParams{
		Protocol: protocol,
		ID:       agendaPointID,
	}); err != nil {
		return fmt.Errorf("update agenda point protocol: %w", err)
	}
	return nil
}

// CreateAttachment links a blob to an agenda point and returns the created record.
func (r *Repository) CreateAttachment(ctx context.Context, agendaPointID, blobID int64, label *string) (*model.AgendaAttachment, error) {
	var nullLabel sql.NullString
	if label != nil {
		nullLabel = sql.NullString{String: *label, Valid: true}
	}
	a, err := r.Queries.CreateAgendaAttachment(ctx, client.CreateAgendaAttachmentParams{
		AgendaPointID: agendaPointID,
		BlobID:        blobID,
		Label:         nullLabel,
	})
	if err != nil {
		return nil, fmt.Errorf("create attachment: %w", err)
	}
	return attachmentFromClient(&a), nil
}

// GetAttachmentByID retrieves an agenda_attachment record by its primary key.
func (r *Repository) GetAttachmentByID(ctx context.Context, id int64) (*model.AgendaAttachment, error) {
	a, err := r.Queries.GetAgendaAttachmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("attachment not found")
		}
		return nil, fmt.Errorf("get attachment: %w", err)
	}
	return attachmentFromClient(&a), nil
}

// ListAttachmentsForAgendaPoint returns all attachments for an agenda point ordered by creation time.
func (r *Repository) ListAttachmentsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.AgendaAttachment, error) {
	rows, err := r.Queries.ListAttachmentsForAgendaPoint(ctx, agendaPointID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	result := make([]*model.AgendaAttachment, len(rows))
	for i := range rows {
		result[i] = attachmentFromClient(&rows[i])
	}
	return result, nil
}

// DeleteAttachment removes an agenda_attachment record by ID.
func (r *Repository) DeleteAttachment(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteAgendaAttachment(ctx, id); err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	return nil
}

func attachmentFromClient(a *client.AgendaAttachment) *model.AgendaAttachment {
	createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)
	att := &model.AgendaAttachment{
		ID:            a.ID,
		AgendaPointID: a.AgendaPointID,
		BlobID:        a.BlobID,
		CreatedAt:     createdAt,
	}
	if a.Label.Valid {
		att.Label = &a.Label.String
	}
	return att
}

// CreateBlob inserts a binary_blob metadata row and returns the created record.
func (r *Repository) CreateBlob(ctx context.Context, filename, contentType string, sizeBytes int64, storagePath string) (*model.BinaryBlob, error) {
	b, err := r.Queries.CreateBinaryBlob(ctx, client.CreateBinaryBlobParams{
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
		StoragePath: storagePath,
	})
	if err != nil {
		return nil, fmt.Errorf("create blob: %w", err)
	}
	return blobFromClient(&b), nil
}

// GetBlobByID retrieves a binary_blob record by its primary key.
func (r *Repository) GetBlobByID(ctx context.Context, id int64) (*model.BinaryBlob, error) {
	b, err := r.Queries.GetBinaryBlobByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("blob not found")
		}
		return nil, fmt.Errorf("get blob: %w", err)
	}
	return blobFromClient(&b), nil
}

// DeleteBlob removes a binary_blob record by ID.
func (r *Repository) DeleteBlob(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteBinaryBlob(ctx, id); err != nil {
		return fmt.Errorf("delete blob: %w", err)
	}
	return nil
}

func blobFromClient(b *client.BinaryBlob) *model.BinaryBlob {
	createdAt, _ := time.Parse(time.RFC3339, b.CreatedAt)
	return &model.BinaryBlob{
		ID:          b.ID,
		Filename:    b.Filename,
		ContentType: b.ContentType,
		SizeBytes:   b.SizeBytes,
		StoragePath: b.StoragePath,
		CreatedAt:   createdAt,
	}
}

// CreateMotion inserts a motion row linked to an agenda point and a blob.
func (r *Repository) CreateMotion(ctx context.Context, agendaPointID, blobID int64, title string) (*model.Motion, error) {
	m, err := r.Queries.CreateMotion(ctx, client.CreateMotionParams{
		AgendaPointID: agendaPointID,
		BlobID:        blobID,
		Title:         title,
	})
	if err != nil {
		return nil, fmt.Errorf("create motion: %w", err)
	}
	return motionFromClient(&m), nil
}

// GetMotionByID retrieves a motion by its primary key.
func (r *Repository) GetMotionByID(ctx context.Context, id int64) (*model.Motion, error) {
	m, err := r.Queries.GetMotionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("motion not found")
		}
		return nil, fmt.Errorf("get motion: %w", err)
	}
	return motionFromClient(&m), nil
}

// ListMotionsForAgendaPoint returns all motions for an agenda point ordered by creation time.
func (r *Repository) ListMotionsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.Motion, error) {
	rows, err := r.Queries.ListMotionsForAgendaPoint(ctx, agendaPointID)
	if err != nil {
		return nil, fmt.Errorf("list motions: %w", err)
	}
	result := make([]*model.Motion, len(rows))
	for i := range rows {
		result[i] = motionFromClient(&rows[i])
	}
	return result, nil
}

// DeleteMotion removes a motion by ID.
func (r *Repository) DeleteMotion(ctx context.Context, id int64) error {
	if err := r.Queries.DeleteMotion(ctx, id); err != nil {
		return fmt.Errorf("delete motion: %w", err)
	}
	return nil
}

// SetMotionVotes records the vote tally for a motion.
func (r *Repository) SetMotionVotes(ctx context.Context, id, votesFor, votesAgainst, votesAbstained, votesEligible int64) error {
	if err := r.Queries.SetMotionVotes(ctx, client.SetMotionVotesParams{
		ID:             id,
		VotesFor:       sql.NullInt64{Int64: votesFor, Valid: true},
		VotesAgainst:   sql.NullInt64{Int64: votesAgainst, Valid: true},
		VotesAbstained: sql.NullInt64{Int64: votesAbstained, Valid: true},
		VotesEligible:  sql.NullInt64{Int64: votesEligible, Valid: true},
	}); err != nil {
		return fmt.Errorf("set motion votes: %w", err)
	}
	return nil
}

func motionFromClient(m *client.Motion) *model.Motion {
	createdAt, _ := time.Parse(time.RFC3339, m.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, m.UpdatedAt)
	motion := &model.Motion{
		ID:            m.ID,
		AgendaPointID: m.AgendaPointID,
		BlobID:        m.BlobID,
		Title:         m.Title,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	if m.VotesFor.Valid {
		motion.VotesFor = &m.VotesFor.Int64
	}
	if m.VotesAgainst.Valid {
		motion.VotesAgainst = &m.VotesAgainst.Int64
	}
	if m.VotesAbstained.Valid {
		motion.VotesAbstained = &m.VotesAbstained.Int64
	}
	if m.VotesEligible.Valid {
		motion.VotesEligible = &m.VotesEligible.Int64
	}
	return motion
}
