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

// CreateOAuthAccount creates a new sitewide account with auth_method='oauth' and no password credential.
func (r *Repository) CreateOAuthAccount(ctx context.Context, username, fullName string) (*model.Account, error) {
	var (
		id         int64
		outUser    string
		authMethod string
		isAdmin    bool
		createdAt  string
		updatedAt  string
		fullNameNS sql.NullString
	)
	if err := r.DB.QueryRowContext(
		ctx,
		`INSERT INTO accounts (username, full_name, auth_method, created_at, updated_at)
		 VALUES (?, ?, 'oauth', datetime('now'), datetime('now'))
		 RETURNING id, username, auth_method, is_admin, created_at, updated_at, full_name`,
		username,
		sql.NullString{String: fullName, Valid: strings.TrimSpace(fullName) != ""},
	).Scan(&id, &outUser, &authMethod, &isAdmin, &createdAt, &updatedAt, &fullNameNS); err != nil {
		return nil, fmt.Errorf("create oauth account: %w", err)
	}
	created, _ := time.Parse(time.RFC3339, createdAt)
	updated, _ := time.Parse(time.RFC3339, updatedAt)
	return &model.Account{
		ID:         id,
		Username:   outUser,
		FullName:   nullStringValue(fullNameNS, outUser),
		AuthMethod: authMethod,
		IsAdmin:    isAdmin,
		CreatedAt:  created,
		UpdatedAt:  updated,
	}, nil
}

// GetOAuthIdentityByIssuerSubject retrieves an OAuth identity mapping by issuer and subject.
func (r *Repository) GetOAuthIdentityByIssuerSubject(ctx context.Context, issuer, subject string) (*model.OAuthIdentity, error) {
	row := r.DB.QueryRowContext(
		ctx,
		`SELECT id, issuer, subject, account_id, username, full_name, email, groups_json, created_at, updated_at
		 FROM oauth_identities
		 WHERE issuer = ? AND subject = ?`,
		issuer, subject,
	)
	var (
		m         model.OAuthIdentity
		username  sql.NullString
		fullName  sql.NullString
		email     sql.NullString
		groups    sql.NullString
		createdAt string
		updatedAt string
	)
	if err := row.Scan(
		&m.ID,
		&m.Issuer,
		&m.Subject,
		&m.AccountID,
		&username,
		&fullName,
		&email,
		&groups,
		&createdAt,
		&updatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("oauth identity not found")
		}
		return nil, fmt.Errorf("get oauth identity: %w", err)
	}
	if username.Valid {
		m.Username = &username.String
	}
	if fullName.Valid {
		m.FullName = &fullName.String
	}
	if email.Valid {
		m.Email = &email.String
	}
	if groups.Valid {
		m.GroupsJSON = &groups.String
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &m, nil
}

// UpsertOAuthIdentity creates or updates an OAuth identity mapping.
func (r *Repository) UpsertOAuthIdentity(
	ctx context.Context,
	issuer, subject string,
	accountID int64,
	username, fullName, email *string,
	groupsJSON *string,
) (*model.OAuthIdentity, error) {
	toNull := func(v *string) sql.NullString {
		if v == nil {
			return sql.NullString{}
		}
		return sql.NullString{String: *v, Valid: true}
	}
	if _, err := r.DB.ExecContext(
		ctx,
		`INSERT INTO oauth_identities (
		     issuer, subject, account_id, username, full_name, email, groups_json, created_at, updated_at
		 )
		 VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		 ON CONFLICT (issuer, subject) DO UPDATE
		 SET account_id = excluded.account_id,
		     username = excluded.username,
		     full_name = excluded.full_name,
		     email = excluded.email,
		     groups_json = excluded.groups_json,
		     updated_at = datetime('now')`,
		issuer,
		subject,
		accountID,
		toNull(username),
		toNull(fullName),
		toNull(email),
		toNull(groupsJSON),
	); err != nil {
		return nil, fmt.Errorf("upsert oauth identity: %w", err)
	}
	return r.GetOAuthIdentityByIssuerSubject(ctx, issuer, subject)
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

// SyncOAuthCommitteeMemberships applies OAuth-derived committee access and role updates.
// It performs full sync for OAuth-managed memberships and leaves manual memberships unchanged.
func (r *Repository) SyncOAuthCommitteeMemberships(ctx context.Context, accountID int64, desired []model.OAuthDesiredMembership) error {
	type existingMembership struct {
		UserID       int64
		CommitteeID  int64
		Role         string
		Quoted       bool
		OAuthManaged bool
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sync oauth memberships begin tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(
		ctx,
		`SELECT u.id, u.committee_id, u.role, u.quoted,
		        CASE WHEN om.user_id IS NULL THEN 0 ELSE 1 END AS oauth_managed
		   FROM users u
		   LEFT JOIN oauth_managed_memberships om ON om.user_id = u.id
		  WHERE u.account_id = ?`,
		accountID,
	)
	if err != nil {
		return fmt.Errorf("sync oauth memberships list existing: %w", err)
	}
	defer rows.Close()

	byCommittee := map[int64]existingMembership{}
	for rows.Next() {
		var m existingMembership
		var oauthManaged int64
		if err := rows.Scan(&m.UserID, &m.CommitteeID, &m.Role, &m.Quoted, &oauthManaged); err != nil {
			return fmt.Errorf("sync oauth memberships scan existing: %w", err)
		}
		m.OAuthManaged = oauthManaged == 1
		byCommittee[m.CommitteeID] = m
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("sync oauth memberships iterate existing: %w", err)
	}

	desiredMap := make(map[int64]string, len(desired))
	for _, d := range desired {
		currentRole, exists := desiredMap[d.CommitteeID]
		if !exists {
			desiredMap[d.CommitteeID] = d.Role
			continue
		}
		if d.Role == "chairperson" || currentRole != "chairperson" {
			desiredMap[d.CommitteeID] = d.Role
		}
	}

	for committeeID, role := range desiredMap {
		current, exists := byCommittee[committeeID]
		if !exists {
			var userID int64
			if err := tx.QueryRowContext(
				ctx,
				`INSERT INTO users (account_id, committee_id, role, quoted, created_at, updated_at)
				 VALUES (?, ?, ?, 0, datetime('now'), datetime('now'))
				 RETURNING id`,
				accountID, committeeID, role,
			).Scan(&userID); err != nil {
				return fmt.Errorf("sync oauth memberships create membership: %w", err)
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO oauth_managed_memberships (user_id, last_synced_at)
				 VALUES (?, datetime('now'))
				 ON CONFLICT (user_id) DO UPDATE SET last_synced_at = datetime('now')`,
				userID,
			); err != nil {
				return fmt.Errorf("sync oauth memberships mark managed: %w", err)
			}
			continue
		}
		if current.OAuthManaged {
			if current.Role != role {
				if _, err := tx.ExecContext(
					ctx,
					`UPDATE users SET role = ?, updated_at = datetime('now') WHERE id = ?`,
					role, current.UserID,
				); err != nil {
					return fmt.Errorf("sync oauth memberships update role: %w", err)
				}
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO oauth_managed_memberships (user_id, last_synced_at)
				 VALUES (?, datetime('now'))
				 ON CONFLICT (user_id) DO UPDATE SET last_synced_at = datetime('now')`,
				current.UserID,
			); err != nil {
				return fmt.Errorf("sync oauth memberships refresh managed marker: %w", err)
			}
			continue
		}
		// Manual memberships are not OAuth-managed, but we still raise their role if OAuth
		// currently grants a higher permission for the same committee.
		if oauthRoleRank(role) > oauthRoleRank(current.Role) {
			if _, err := tx.ExecContext(
				ctx,
				`UPDATE users SET role = ?, updated_at = datetime('now') WHERE id = ?`,
				role, current.UserID,
			); err != nil {
				return fmt.Errorf("sync oauth memberships promote manual role: %w", err)
			}
		}
	}

	for committeeID, current := range byCommittee {
		if !current.OAuthManaged {
			continue
		}
		if _, stillDesired := desiredMap[committeeID]; stillDesired {
			continue
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, current.UserID); err != nil {
			return fmt.Errorf("sync oauth memberships delete stale membership: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sync oauth memberships commit tx: %w", err)
	}
	return nil
}

func oauthRoleRank(role string) int {
	switch role {
	case "chairperson":
		return 2
	case "member":
		return 1
	default:
		return 0
	}
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
		FullName:   nullStringValue(a.FullName, a.Username),
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
		FullName:    nullStringValue(r.FullName, r.Username),
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
		FullName:    nullStringValue(r.FullName, r.Username),
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
		FullName:      nullStringValue(r.FullName, r.Username),
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
		ID:           r.ID,
		AccountID:    r.AccountID,
		CommitteeID:  r.CommitteeID,
		Username:     r.Username,
		FullName:     nullStringValue(r.FullName, r.Username),
		Quoted:       r.Quoted,
		Role:         r.Role,
		OAuthManaged: r.OauthManaged == 1,
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
		account, err = r.Queries.CreateAccount(ctx, client.CreateAccountParams{
			Username: username,
			FullName: sql.NullString{String: fullName, Valid: fullName != ""},
		})
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

// CreateAccount creates a new sitewide account with credentials.
func (r *Repository) CreateAccount(ctx context.Context, username, fullName, passwordHash string) (*model.Account, error) {
	account, err := r.Queries.CreateAccount(ctx, client.CreateAccountParams{
		Username: username,
		FullName: sql.NullString{String: fullName, Valid: fullName != ""},
	})
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

// AssignAccountToCommittee creates a committee membership for an existing account.
func (r *Repository) AssignAccountToCommittee(ctx context.Context, committeeID, accountID int64, quoted bool, role string) error {
	_, err := r.Queries.CreateMembership(ctx, client.CreateMembershipParams{
		AccountID:   accountID,
		CommitteeID: committeeID,
		Role:        role,
		Quoted:      quoted,
	})
	if err != nil {
		return fmt.Errorf("assign account to committee: %w", err)
	}
	return nil
}

// UpdateUserMembership updates quoted/role for an existing membership row.
func (r *Repository) UpdateUserMembership(ctx context.Context, userID int64, quoted bool, role string) error {
	err := r.Queries.UpdateMembership(ctx, client.UpdateMembershipParams{
		Quoted: quoted,
		Role:   role,
		ID:     userID,
	})
	if err != nil {
		return fmt.Errorf("update user membership: %w", err)
	}
	return nil
}

// IsOAuthManagedMembership returns true when a user membership is OAuth-managed.
func (r *Repository) IsOAuthManagedMembership(ctx context.Context, userID int64) (bool, error) {
	var exists int64
	if err := r.DB.QueryRowContext(
		ctx,
		`SELECT CASE WHEN EXISTS (
			SELECT 1 FROM oauth_managed_memberships WHERE user_id = ?
		) THEN 1 ELSE 0 END`,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check oauth managed membership: %w", err)
	}
	return exists == 1, nil
}

// CountAllAccounts returns the total number of accounts.
func (r *Repository) CountAllAccounts(ctx context.Context) (int64, error) {
	count, err := r.Queries.CountAllAccounts(ctx)
	if err != nil {
		return 0, fmt.Errorf("count all accounts: %w", err)
	}
	return count, nil
}

// ListAllAccounts returns a paginated list of accounts.
func (r *Repository) ListAllAccounts(ctx context.Context, limit, offset int) ([]*model.Account, error) {
	accounts, err := r.Queries.ListAllAccounts(ctx, client.ListAllAccountsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list all accounts: %w", err)
	}
	result := make([]*model.Account, len(accounts))
	for i := range accounts {
		result[i] = accountFromClient(&accounts[i])
	}
	return result, nil
}

// ListUnassignedAccountsForCommittee returns accounts without membership in the committee.
func (r *Repository) ListUnassignedAccountsForCommittee(ctx context.Context, committeeID int64) ([]*model.Account, error) {
	accounts, err := r.Queries.ListUnassignedAccountsForCommittee(ctx, committeeID)
	if err != nil {
		return nil, fmt.Errorf("list unassigned accounts for committee: %w", err)
	}
	result := make([]*model.Account, len(accounts))
	for i := range accounts {
		result[i] = accountFromClient(&accounts[i])
	}
	return result, nil
}

// ListOAuthCommitteeGroupRulesByCommitteeSlug lists OAuth group-to-role mappings for one committee.
func (r *Repository) ListOAuthCommitteeGroupRulesByCommitteeSlug(ctx context.Context, slug string) ([]*model.OAuthCommitteeGroupRule, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT r.id, r.committee_id, c.slug, r.group_name, r.role, r.created_at, r.updated_at
		   FROM oauth_committee_group_rules r
		   JOIN committees c ON c.id = r.committee_id
		  WHERE c.slug = ?
		  ORDER BY r.group_name`,
		slug,
	)
	if err != nil {
		return nil, fmt.Errorf("list oauth committee group rules by slug: %w", err)
	}
	defer rows.Close()

	var result []*model.OAuthCommitteeGroupRule
	for rows.Next() {
		var (
			item      model.OAuthCommitteeGroupRule
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(
			&item.ID,
			&item.CommitteeID,
			&item.CommitteeSlug,
			&item.GroupName,
			&item.Role,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan oauth committee group rule by slug: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		result = append(result, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth committee group rules by slug: %w", err)
	}
	return result, nil
}

// ListAllOAuthCommitteeGroupRules lists all OAuth group rules across committees.
func (r *Repository) ListAllOAuthCommitteeGroupRules(ctx context.Context) ([]*model.OAuthCommitteeGroupRule, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		`SELECT r.id, r.committee_id, c.slug, r.group_name, r.role, r.created_at, r.updated_at
		   FROM oauth_committee_group_rules r
		   JOIN committees c ON c.id = r.committee_id
		  ORDER BY r.committee_id, r.group_name`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all oauth committee group rules: %w", err)
	}
	defer rows.Close()

	var result []*model.OAuthCommitteeGroupRule
	for rows.Next() {
		var (
			item      model.OAuthCommitteeGroupRule
			createdAt string
			updatedAt string
		)
		if err := rows.Scan(
			&item.ID,
			&item.CommitteeID,
			&item.CommitteeSlug,
			&item.GroupName,
			&item.Role,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan oauth committee group rule: %w", err)
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		result = append(result, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth committee group rules: %w", err)
	}
	return result, nil
}

// CreateOAuthCommitteeGroupRuleByCommitteeSlug creates a group rule for a committee.
func (r *Repository) CreateOAuthCommitteeGroupRuleByCommitteeSlug(ctx context.Context, slug, groupName, role string) (*model.OAuthCommitteeGroupRule, error) {
	var (
		item      model.OAuthCommitteeGroupRule
		createdAt string
		updatedAt string
	)
	if err := r.DB.QueryRowContext(
		ctx,
		`INSERT INTO oauth_committee_group_rules (
		     committee_id, group_name, role, created_at, updated_at
		 )
		 SELECT c.id, ?, ?, datetime('now'), datetime('now')
		 FROM committees c
		 WHERE c.slug = ?
		 RETURNING id, committee_id, group_name, role, created_at, updated_at`,
		groupName, role, slug,
	).Scan(
		&item.ID,
		&item.CommitteeID,
		&item.GroupName,
		&item.Role,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, fmt.Errorf("create oauth committee group rule: %w", err)
	}
	item.CommitteeSlug = slug
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &item, nil
}

// DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug deletes a group rule scoped to a committee slug.
func (r *Repository) DeleteOAuthCommitteeGroupRuleByIDAndCommitteeSlug(ctx context.Context, id int64, slug string) error {
	res, err := r.DB.ExecContext(
		ctx,
		`DELETE FROM oauth_committee_group_rules
		  WHERE id = ?
		    AND committee_id = (SELECT c.id FROM committees c WHERE c.slug = ?)`,
		id, slug,
	)
	if err != nil {
		return fmt.Errorf("delete oauth committee group rule: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("oauth committee group rule not found")
	}
	return nil
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

// SetActiveMeeting sets or clears the current active meeting for a committee.
func (r *Repository) SetActiveMeeting(ctx context.Context, slug string, meetingID *int64) error {
	var activeID sql.NullInt64
	if meetingID != nil {
		activeID = sql.NullInt64{Int64: *meetingID, Valid: true}
	}
	err := r.Queries.SetActiveMeeting(ctx, client.SetActiveMeetingParams{
		CurrentMeetingID: activeID,
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
	return &model.Meeting{
		ID:                           m.ID,
		Name:                         m.Name,
		Description:                  m.Description,
		Secret:                       m.Secret,
		SignupOpen:                   m.SignupOpen,
		CurrentAgendaPointID:         currentAgendaPointID,
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

func nullStringValue(n sql.NullString, fallback string) string {
	if n.Valid && n.String != "" {
		return n.String
	}
	return fallback
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
		CurrentSpeakerID:             currentSpeakerID,
		GenderQuotationEnabled:       nullBoolToPtr(ap.GenderQuotationEnabled),
		FirstSpeakerQuotationEnabled: nullBoolToPtr(ap.FirstSpeakerQuotationEnabled),
		ModeratorID:                  nullInt64ToPtr(ap.ModeratorID),
		CurrentAttachmentID:          nullInt64ToPtr(ap.CurrentAttachmentID),
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

// ListSubAgendaPointsForParent returns child agenda points for one parent.
func (r *Repository) ListSubAgendaPointsForParent(ctx context.Context, meetingID, parentID int64) ([]*model.AgendaPoint, error) {
	rows, err := r.Queries.ListSubAgendaPointsForParent(ctx, client.ListSubAgendaPointsForParentParams{
		MeetingID: meetingID,
		ParentID:  sql.NullInt64{Int64: parentID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list sub-agenda points for parent: %w", err)
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

func (r *Repository) moveAgendaPoint(ctx context.Context, meetingID, agendaPointID int64, direction int) error {
	ap, err := r.GetAgendaPointByID(ctx, agendaPointID)
	if err != nil {
		return err
	}
	if ap.MeetingID != meetingID {
		return fmt.Errorf("agenda point does not belong to meeting")
	}

	var siblings []*model.AgendaPoint
	if ap.ParentID == nil {
		siblings, err = r.ListAgendaPointsForMeeting(ctx, meetingID)
	} else {
		siblings, err = r.ListSubAgendaPointsForParent(ctx, meetingID, *ap.ParentID)
	}
	if err != nil {
		return err
	}

	idx := -1
	for i, s := range siblings {
		if s.ID == agendaPointID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("agenda point not found in siblings")
	}
	targetIdx := idx + direction
	if targetIdx < 0 || targetIdx >= len(siblings) {
		return nil
	}

	a := siblings[idx]
	b := siblings[targetIdx]

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	if err := qtx.SetAgendaPointPosition(ctx, client.SetAgendaPointPositionParams{
		Position: -1,
		ID:       a.ID,
	}); err != nil {
		return fmt.Errorf("set temporary position: %w", err)
	}
	if err := qtx.SetAgendaPointPosition(ctx, client.SetAgendaPointPositionParams{
		Position: a.Position,
		ID:       b.ID,
	}); err != nil {
		return fmt.Errorf("set swapped position: %w", err)
	}
	if err := qtx.SetAgendaPointPosition(ctx, client.SetAgendaPointPositionParams{
		Position: b.Position,
		ID:       a.ID,
	}); err != nil {
		return fmt.Errorf("finalize swapped position: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// MoveAgendaPointUp swaps an agenda point with the previous sibling.
func (r *Repository) MoveAgendaPointUp(ctx context.Context, meetingID, agendaPointID int64) error {
	return r.moveAgendaPoint(ctx, meetingID, agendaPointID, -1)
}

// MoveAgendaPointDown swaps an agenda point with the next sibling.
func (r *Repository) MoveAgendaPointDown(ctx context.Context, meetingID, agendaPointID int64) error {
	return r.moveAgendaPoint(ctx, meetingID, agendaPointID, 1)
}

// ApplyAgendaPoints transactionally updates agenda structure to the desired points.
func (r *Repository) ApplyAgendaPoints(ctx context.Context, meetingID int64, points []repository.AgendaApplyPoint, deleteIDs []int64) error {
	if len(points) == 0 {
		return fmt.Errorf("no agenda points provided")
	}

	keySet := make(map[string]struct{}, len(points))
	for _, p := range points {
		if strings.TrimSpace(p.Key) == "" {
			return fmt.Errorf("agenda point key is required")
		}
		if _, exists := keySet[p.Key]; exists {
			return fmt.Errorf("duplicate agenda point key: %s", p.Key)
		}
		keySet[p.Key] = struct{}{}
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)

	if err := qtx.BumpAgendaPointPositionsForMeeting(ctx, meetingID); err != nil {
		return fmt.Errorf("bump agenda positions: %w", err)
	}

	for _, id := range deleteIDs {
		if err := qtx.DeleteAgendaPoint(ctx, id); err != nil {
			return fmt.Errorf("delete agenda point %d: %w", id, err)
		}
	}

	keyToID := make(map[string]int64, len(points))

	topLevel := make([]repository.AgendaApplyPoint, 0, len(points))
	children := make([]repository.AgendaApplyPoint, 0, len(points))
	for _, p := range points {
		if p.ParentKey == nil {
			topLevel = append(topLevel, p)
		} else {
			children = append(children, p)
		}
	}
	sort.SliceStable(topLevel, func(i, j int) bool {
		if topLevel[i].Position != topLevel[j].Position {
			return topLevel[i].Position < topLevel[j].Position
		}
		return topLevel[i].Key < topLevel[j].Key
	})
	sort.SliceStable(children, func(i, j int) bool {
		if *children[i].ParentKey != *children[j].ParentKey {
			return *children[i].ParentKey < *children[j].ParentKey
		}
		if children[i].Position != children[j].Position {
			return children[i].Position < children[j].Position
		}
		return children[i].Key < children[j].Key
	})

	updatePoint := func(p repository.AgendaApplyPoint, parentID *int64) (int64, error) {
		var pid sql.NullInt64
		if parentID != nil {
			pid = sql.NullInt64{Int64: *parentID, Valid: true}
		}
		if p.ExistingID != nil {
			if err := qtx.UpdateAgendaPointStructure(ctx, client.UpdateAgendaPointStructureParams{
				ParentID:  pid,
				Position:  p.Position,
				Title:     p.Title,
				ID:        *p.ExistingID,
				MeetingID: meetingID,
			}); err != nil {
				return 0, err
			}
			return *p.ExistingID, nil
		}
		if parentID == nil {
			created, err := qtx.CreateAgendaPoint(ctx, client.CreateAgendaPointParams{
				MeetingID: meetingID,
				Position:  p.Position,
				Title:     p.Title,
			})
			if err != nil {
				return 0, err
			}
			return created.ID, nil
		}
		created, err := qtx.CreateSubAgendaPoint(ctx, client.CreateSubAgendaPointParams{
			MeetingID: meetingID,
			ParentID:  pid,
			Position:  p.Position,
			Title:     p.Title,
		})
		if err != nil {
			return 0, err
		}
		return created.ID, nil
	}

	for _, p := range topLevel {
		id, err := updatePoint(p, nil)
		if err != nil {
			return fmt.Errorf("upsert top-level agenda point: %w", err)
		}
		keyToID[p.Key] = id
	}
	for _, p := range children {
		parentID, ok := keyToID[*p.ParentKey]
		if !ok {
			return fmt.Errorf("missing parent key: %s", *p.ParentKey)
		}
		id, err := updatePoint(p, &parentID)
		if err != nil {
			return fmt.Errorf("upsert child agenda point: %w", err)
		}
		keyToID[p.Key] = id
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
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

// SetCurrentAttachment sets an agenda point's current attachment.
func (r *Repository) SetCurrentAttachment(ctx context.Context, agendaPointID, attachmentID int64) error {
	if err := r.Queries.SetCurrentAttachment(ctx, client.SetCurrentAttachmentParams{
		CurrentAttachmentID: sql.NullInt64{Int64: attachmentID, Valid: true},
		ID:                  agendaPointID,
	}); err != nil {
		return fmt.Errorf("set current attachment: %w", err)
	}
	return nil
}

// ClearCurrentDocument clears current attachment on an agenda point.
func (r *Repository) ClearCurrentDocument(ctx context.Context, agendaPointID int64) error {
	if err := r.Queries.ClearCurrentDocument(ctx, agendaPointID); err != nil {
		return fmt.Errorf("clear current document: %w", err)
	}
	return nil
}

// AddSpeaker adds an attendee to the speakers list for an agenda point.
func (r *Repository) AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error) {
	// Point-of-order entries never receive the first-speaker flag.
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

// CreateVoteDefinition inserts a new vote definition in draft state.
func (r *Repository) CreateVoteDefinition(
	ctx context.Context,
	meetingID, agendaPointID int64,
	name, visibility string,
	minSelections, maxSelections int64,
) (*model.VoteDefinition, error) {
	row, err := r.Queries.CreateVoteDefinition(ctx, client.CreateVoteDefinitionParams{
		MeetingID:     meetingID,
		AgendaPointID: agendaPointID,
		Name:          name,
		Visibility:    visibility,
		MinSelections: minSelections,
		MaxSelections: maxSelections,
	})
	if err != nil {
		return nil, fmt.Errorf("create vote definition: %w", err)
	}
	return voteDefinitionFromClient(&row), nil
}

// UpdateVoteDefinitionDraft updates a draft vote definition.
func (r *Repository) UpdateVoteDefinitionDraft(
	ctx context.Context,
	id int64,
	meetingID, agendaPointID int64,
	name, visibility string,
	minSelections, maxSelections int64,
) (*model.VoteDefinition, error) {
	row, err := r.Queries.UpdateVoteDefinitionDraft(ctx, client.UpdateVoteDefinitionDraftParams{
		MeetingID:     meetingID,
		AgendaPointID: agendaPointID,
		Name:          name,
		Visibility:    visibility,
		MinSelections: minSelections,
		MaxSelections: maxSelections,
		ID:            id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not in draft state")
		}
		return nil, fmt.Errorf("update vote definition draft: %w", err)
	}
	return voteDefinitionFromClient(&row), nil
}

// OpenVoteWithEligibleVoters atomically snapshots eligible voters and opens a vote.
func (r *Repository) OpenVoteWithEligibleVoters(ctx context.Context, voteDefinitionID int64, attendeeIDs []int64) (*model.VoteDefinition, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("open vote begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	vd, err := qtx.GetVoteDefinitionByID(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State != model.VoteStateDraft {
		return nil, fmt.Errorf("vote definition not in draft state")
	}

	existingEligible, err := qtx.CountEligibleVotersForVoteDefinition(ctx, voteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("count existing eligible voters: %w", err)
	}
	if existingEligible > 0 {
		return nil, fmt.Errorf("eligible voters already exist for vote")
	}

	seen := make(map[int64]struct{}, len(attendeeIDs))
	for _, attendeeID := range attendeeIDs {
		if _, ok := seen[attendeeID]; ok {
			continue
		}
		seen[attendeeID] = struct{}{}
		if err := qtx.InsertEligibleVoter(ctx, client.InsertEligibleVoterParams{
			VoteDefinitionID: voteDefinitionID,
			MeetingID:        vd.MeetingID,
			AttendeeID:       attendeeID,
		}); err != nil {
			return nil, fmt.Errorf("insert eligible voter %d: %w", attendeeID, err)
		}
	}

	opened, err := qtx.SetVoteDefinitionOpen(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition cannot be opened")
		}
		return nil, fmt.Errorf("set vote definition open: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("open vote commit tx: %w", err)
	}
	return voteDefinitionFromClient(&opened), nil
}

// CloseVote transitions an open/counting vote according to visibility and outstanding secret ballots.
func (r *Repository) CloseVote(ctx context.Context, voteDefinitionID int64) (*model.CloseVoteResult, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("close vote begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	vd, err := qtx.GetVoteDefinitionByID(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("load vote definition: %w", err)
	}

	transitionToClosedFrom := func(state string) (*model.CloseVoteResult, error) {
		var closed client.VoteDefinition
		if state == model.VoteStateOpen {
			closed, err = qtx.SetVoteDefinitionClosedFromOpen(ctx, voteDefinitionID)
		} else {
			closed, err = qtx.SetVoteDefinitionClosedFromCounting(ctx, voteDefinitionID)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, model.VoteCloseStateError{State: state}
			}
			return nil, fmt.Errorf("set vote definition closed: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("close vote commit tx: %w", err)
		}
		return &model.CloseVoteResult{
			Vote:    voteDefinitionFromClient(&closed),
			Outcome: model.CloseVoteOutcomeClosed,
		}, nil
	}

	switch vd.State {
	case model.VoteStateOpen:
		if vd.Visibility == model.VoteVisibilitySecret {
			castCount, err := qtx.CountVoteCastsForVoteDefinition(ctx, voteDefinitionID)
			if err != nil {
				return nil, fmt.Errorf("count vote casts: %w", err)
			}
			secretBallotCount, err := qtx.CountSecretVoteBallots(ctx, voteDefinitionID)
			if err != nil {
				return nil, fmt.Errorf("count secret vote ballots: %w", err)
			}

			if castCount > secretBallotCount {
				counting, err := qtx.SetVoteDefinitionCountingFromOpen(ctx, voteDefinitionID)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return nil, model.VoteCloseStateError{State: vd.State}
					}
					return nil, fmt.Errorf("set vote definition counting: %w", err)
				}
				if err := tx.Commit(); err != nil {
					return nil, fmt.Errorf("close vote commit tx: %w", err)
				}
				return &model.CloseVoteResult{
					Vote:    voteDefinitionFromClient(&counting),
					Outcome: model.CloseVoteOutcomeEnteredCounting,
				}, nil
			}
		}
		return transitionToClosedFrom(model.VoteStateOpen)
	case model.VoteStateCounting:
		castCount, err := qtx.CountVoteCastsForVoteDefinition(ctx, voteDefinitionID)
		if err != nil {
			return nil, fmt.Errorf("count vote casts: %w", err)
		}
		secretBallotCount, err := qtx.CountSecretVoteBallots(ctx, voteDefinitionID)
		if err != nil {
			return nil, fmt.Errorf("count secret vote ballots: %w", err)
		}
		if castCount > secretBallotCount {
			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("close vote commit tx: %w", err)
			}
			return &model.CloseVoteResult{
				Vote:    voteDefinitionFromClient(&vd),
				Outcome: model.CloseVoteOutcomeStillCounting,
			}, nil
		}
		return transitionToClosedFrom(model.VoteStateCounting)
	case model.VoteStateClosed, model.VoteStateArchived:
		return nil, model.VoteCloseStateError{State: vd.State}
	default:
		return nil, model.VoteCloseStateError{State: vd.State}
	}
}

// ArchiveVote transitions a closed vote definition to archived.
func (r *Repository) ArchiveVote(ctx context.Context, voteDefinitionID int64) (*model.VoteDefinition, error) {
	row, err := r.Queries.SetVoteDefinitionArchived(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not closed")
		}
		return nil, fmt.Errorf("archive vote definition: %w", err)
	}
	return voteDefinitionFromClient(&row), nil
}

// GetVoteDefinitionByID returns one vote definition.
func (r *Repository) GetVoteDefinitionByID(ctx context.Context, id int64) (*model.VoteDefinition, error) {
	row, err := r.Queries.GetVoteDefinitionByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("get vote definition: %w", err)
	}
	return voteDefinitionFromClient(&row), nil
}

// ListVoteDefinitionsForAgendaPoint lists vote definitions for one agenda point.
func (r *Repository) ListVoteDefinitionsForAgendaPoint(ctx context.Context, agendaPointID int64) ([]*model.VoteDefinition, error) {
	rows, err := r.Queries.ListVoteDefinitionsForAgendaPoint(ctx, agendaPointID)
	if err != nil {
		return nil, fmt.Errorf("list vote definitions: %w", err)
	}
	result := make([]*model.VoteDefinition, len(rows))
	for i := range rows {
		result[i] = voteDefinitionFromClient(&rows[i])
	}
	return result, nil
}

// ReplaceVoteOptions overwrites vote options for a draft vote definition.
func (r *Repository) ReplaceVoteOptions(ctx context.Context, voteDefinitionID int64, options []repository.VoteOptionInput) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("replace vote options begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	vd, err := qtx.GetVoteDefinitionByID(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("vote definition not found")
		}
		return fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State != model.VoteStateDraft {
		return fmt.Errorf("vote definition not in draft state")
	}

	if err := qtx.DeleteVoteOptionsForVoteDefinition(ctx, voteDefinitionID); err != nil {
		return fmt.Errorf("delete previous vote options: %w", err)
	}

	for _, option := range options {
		if _, err := qtx.CreateVoteOption(ctx, client.CreateVoteOptionParams{
			VoteDefinitionID: voteDefinitionID,
			Label:            option.Label,
			Position:         option.Position,
		}); err != nil {
			return fmt.Errorf("create vote option: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("replace vote options commit tx: %w", err)
	}
	return nil
}

// ListVoteOptions lists all options for one vote definition.
func (r *Repository) ListVoteOptions(ctx context.Context, voteDefinitionID int64) ([]*model.VoteOption, error) {
	rows, err := r.Queries.ListVoteOptionsForVoteDefinition(ctx, voteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("list vote options: %w", err)
	}
	result := make([]*model.VoteOption, len(rows))
	for i := range rows {
		result[i] = voteOptionFromClient(&rows[i])
	}
	return result, nil
}

// ListEligibleVoters lists the eligibility snapshot for a vote definition.
func (r *Repository) ListEligibleVoters(ctx context.Context, voteDefinitionID int64) ([]*model.EligibleVoter, error) {
	rows, err := r.Queries.ListEligibleVotersForVoteDefinition(ctx, voteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("list eligible voters: %w", err)
	}
	result := make([]*model.EligibleVoter, len(rows))
	for i := range rows {
		result[i] = eligibleVoterFromClient(&rows[i])
	}
	return result, nil
}

// RegisterVoteCast records that one eligible attendee has cast a vote.
func (r *Repository) RegisterVoteCast(ctx context.Context, voteDefinitionID, meetingID, attendeeID int64, source string) (*model.VoteCast, error) {
	vd, err := r.Queries.GetVoteDefinitionByID(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State != model.VoteStateOpen {
		return nil, fmt.Errorf("vote definition not open")
	}
	if vd.MeetingID != meetingID {
		return nil, fmt.Errorf("meeting mismatch for vote definition")
	}

	row, err := r.Queries.CreateVoteCast(ctx, client.CreateVoteCastParams{
		VoteDefinitionID: voteDefinitionID,
		MeetingID:        meetingID,
		AttendeeID:       attendeeID,
		Source:           source,
	})
	if err != nil {
		return nil, fmt.Errorf("create vote cast: %w", err)
	}
	return voteCastFromClient(&row), nil
}

// ListVoteCasts lists all cast rows for a vote definition.
func (r *Repository) ListVoteCasts(ctx context.Context, voteDefinitionID int64) ([]*model.VoteCast, error) {
	rows, err := r.Queries.ListVoteCastsForVoteDefinition(ctx, voteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("list vote casts: %w", err)
	}
	result := make([]*model.VoteCast, len(rows))
	for i := range rows {
		result[i] = voteCastFromClient(&rows[i])
	}
	return result, nil
}

// SubmitOpenBallot creates an open ballot and selections in one transaction.
func (r *Repository) SubmitOpenBallot(ctx context.Context, submission repository.OpenBallotSubmission) (*model.VoteBallot, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("submit open ballot begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	vd, err := qtx.GetVoteDefinitionByID(ctx, submission.VoteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State != model.VoteStateOpen {
		return nil, fmt.Errorf("vote definition not open")
	}
	if vd.Visibility != model.VoteVisibilityOpen {
		return nil, fmt.Errorf("vote definition is not open-visibility")
	}
	if vd.MeetingID != submission.MeetingID {
		return nil, fmt.Errorf("meeting mismatch for vote definition")
	}

	if err := validateVoteSelectionInput(ctx, qtx, vd, submission.OptionIDs); err != nil {
		return nil, err
	}

	castRow, err := qtx.GetVoteCastByVoteAndAttendee(ctx, client.GetVoteCastByVoteAndAttendeeParams{
		VoteDefinitionID: submission.VoteDefinitionID,
		AttendeeID:       submission.AttendeeID,
	})
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("load vote cast: %w", err)
		}
		castRow, err = qtx.CreateVoteCast(ctx, client.CreateVoteCastParams{
			VoteDefinitionID: submission.VoteDefinitionID,
			MeetingID:        submission.MeetingID,
			AttendeeID:       submission.AttendeeID,
			Source:           submission.Source,
		})
		if err != nil {
			return nil, fmt.Errorf("create vote cast: %w", err)
		}
	}

	ballotRow, err := qtx.CreateOpenVoteBallot(ctx, client.CreateOpenVoteBallotParams{
		VoteDefinitionID: submission.VoteDefinitionID,
		CastID:           sql.NullInt64{Int64: castRow.ID, Valid: true},
		AttendeeID:       sql.NullInt64{Int64: submission.AttendeeID, Valid: true},
		ReceiptToken:     submission.ReceiptToken,
	})
	if err != nil {
		return nil, fmt.Errorf("create open vote ballot: %w", err)
	}

	for _, optionID := range submission.OptionIDs {
		if err := qtx.CreateVoteBallotSelection(ctx, client.CreateVoteBallotSelectionParams{
			BallotID:         ballotRow.ID,
			VoteDefinitionID: submission.VoteDefinitionID,
			OptionID:         optionID,
		}); err != nil {
			return nil, fmt.Errorf("create vote ballot selection: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("submit open ballot commit tx: %w", err)
	}
	return voteBallotFromClient(&ballotRow), nil
}

// SubmitSecretBallot creates one secret ballot and its selections.
func (r *Repository) SubmitSecretBallot(ctx context.Context, submission repository.SecretBallotSubmission) (*model.VoteBallot, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("submit secret ballot begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := r.Queries.WithTx(tx)
	vd, err := qtx.GetVoteDefinitionByID(ctx, submission.VoteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("vote definition not found")
		}
		return nil, fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State != model.VoteStateOpen && vd.State != model.VoteStateCounting {
		return nil, fmt.Errorf("vote definition not open or counting")
	}
	if vd.Visibility != model.VoteVisibilitySecret {
		return nil, fmt.Errorf("vote definition is not secret-visibility")
	}

	if err := validateVoteSelectionInput(ctx, qtx, vd, submission.OptionIDs); err != nil {
		return nil, err
	}

	secretBallotCount, err := qtx.CountSecretVoteBallots(ctx, submission.VoteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("count secret vote ballots: %w", err)
	}
	castCount, err := qtx.CountVoteCastsForVoteDefinition(ctx, submission.VoteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("count vote casts: %w", err)
	}
	if secretBallotCount >= castCount {
		return nil, fmt.Errorf("cannot create secret ballot without available cast rows")
	}

	ballotRow, err := qtx.CreateSecretVoteBallot(ctx, client.CreateSecretVoteBallotParams{
		VoteDefinitionID:    submission.VoteDefinitionID,
		ReceiptToken:        submission.ReceiptToken,
		EncryptedCommitment: submission.EncryptedCommitment,
		CommitmentCipher:    sql.NullString{String: submission.CommitmentCipher, Valid: true},
		CommitmentVersion:   sql.NullInt64{Int64: submission.CommitmentVersion, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create secret vote ballot: %w", err)
	}

	for _, optionID := range submission.OptionIDs {
		if err := qtx.CreateVoteBallotSelection(ctx, client.CreateVoteBallotSelectionParams{
			BallotID:         ballotRow.ID,
			VoteDefinitionID: submission.VoteDefinitionID,
			OptionID:         optionID,
		}); err != nil {
			return nil, fmt.Errorf("create vote ballot selection: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("submit secret ballot commit tx: %w", err)
	}
	return voteBallotFromClient(&ballotRow), nil
}

// VerifyOpenBallotByReceipt returns open ballot verification data by vote+receipt.
func (r *Repository) VerifyOpenBallotByReceipt(ctx context.Context, voteDefinitionID int64, receiptToken string) (*model.VoteOpenVerification, error) {
	if err := r.ensureVoteResultsReadable(ctx, voteDefinitionID); err != nil {
		return nil, err
	}

	rows, err := r.Queries.GetOpenVoteVerificationRows(ctx, client.GetOpenVoteVerificationRowsParams{
		VoteDefinitionID: voteDefinitionID,
		ReceiptToken:     receiptToken,
	})
	if err != nil {
		return nil, fmt.Errorf("verify open vote ballot: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("open ballot not found")
	}

	first := rows[0]
	if !first.AttendeeNumber.Valid {
		return nil, fmt.Errorf("attendee number missing for verified ballot")
	}

	result := &model.VoteOpenVerification{
		VoteDefinitionID: first.VoteDefinitionID,
		VoteName:         first.VoteName,
		AttendeeID:       first.AttendeeID,
		AttendeeNumber:   first.AttendeeNumber.Int64,
		ReceiptToken:     first.ReceiptToken,
		ChoiceLabels:     make([]string, 0, len(rows)),
		ChoiceOptionIDs:  make([]int64, 0, len(rows)),
	}
	for _, row := range rows {
		if row.OptionID.Valid {
			result.ChoiceOptionIDs = append(result.ChoiceOptionIDs, row.OptionID.Int64)
		}
		if row.OptionLabel.Valid {
			result.ChoiceLabels = append(result.ChoiceLabels, row.OptionLabel.String)
		}
	}
	return result, nil
}

// VerifySecretBallotByReceipt returns secret ballot verification data by vote+receipt.
func (r *Repository) VerifySecretBallotByReceipt(ctx context.Context, voteDefinitionID int64, receiptToken string) (*model.VoteSecretVerification, error) {
	if err := r.ensureVoteResultsReadable(ctx, voteDefinitionID); err != nil {
		return nil, err
	}

	row, err := r.Queries.GetSecretVoteVerification(ctx, client.GetSecretVoteVerificationParams{
		VoteDefinitionID: voteDefinitionID,
		ReceiptToken:     receiptToken,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("secret ballot not found")
		}
		return nil, fmt.Errorf("verify secret vote ballot: %w", err)
	}
	if !row.CommitmentCipher.Valid || !row.CommitmentVersion.Valid {
		return nil, fmt.Errorf("secret ballot commitment metadata incomplete")
	}

	return &model.VoteSecretVerification{
		VoteDefinitionID:    row.VoteDefinitionID,
		VoteName:            row.VoteName,
		ReceiptToken:        row.ReceiptToken,
		EncryptedCommitment: row.EncryptedCommitment,
		CommitmentCipher:    row.CommitmentCipher.String,
		CommitmentVersion:   row.CommitmentVersion.Int64,
	}, nil
}

// GetVoteTallies aggregates per-option ballot counts.
func (r *Repository) GetVoteTallies(ctx context.Context, voteDefinitionID int64) ([]*model.VoteTallyRow, error) {
	if err := r.ensureVoteResultsReadable(ctx, voteDefinitionID); err != nil {
		return nil, err
	}

	rows, err := r.Queries.GetVoteTallies(ctx, voteDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("get vote tallies: %w", err)
	}
	result := make([]*model.VoteTallyRow, len(rows))
	for i := range rows {
		result[i] = &model.VoteTallyRow{
			OptionID: rows[i].OptionID,
			Label:    rows[i].OptionLabel,
			Count:    rows[i].TallyCount,
		}
	}
	return result, nil
}

// GetVoteSubmissionStats returns cast/ballot/eligible counts for one vote definition.
func (r *Repository) GetVoteSubmissionStats(ctx context.Context, voteDefinitionID int64) (*model.VoteSubmissionStats, error) {
	if err := r.ensureVoteResultsReadable(ctx, voteDefinitionID); err != nil {
		return nil, err
	}

	row, err := r.Queries.GetVoteSubmissionStats(ctx, client.GetVoteSubmissionStatsParams{
		VoteDefinitionID:   voteDefinitionID,
		VoteDefinitionID_2: voteDefinitionID,
		VoteDefinitionID_3: voteDefinitionID,
		VoteDefinitionID_4: voteDefinitionID,
		VoteDefinitionID_5: voteDefinitionID,
	})
	if err != nil {
		return nil, fmt.Errorf("get vote submission stats: %w", err)
	}
	return &model.VoteSubmissionStats{
		EligibleCount:     row.EligibleCount,
		CastCount:         row.CastCount,
		BallotCount:       row.BallotCount,
		OpenBallotCount:   row.OpenBallotCount,
		SecretBallotCount: row.SecretBallotCount,
	}, nil
}

// GetVoteSubmissionStatsLive returns cast/ballot/eligible counts for moderator/live progress views.
// Unlike GetVoteSubmissionStats, it is available in all vote states including counting.
func (r *Repository) GetVoteSubmissionStatsLive(ctx context.Context, voteDefinitionID int64) (*model.VoteSubmissionStats, error) {
	row, err := r.Queries.GetVoteSubmissionStats(ctx, client.GetVoteSubmissionStatsParams{
		VoteDefinitionID:   voteDefinitionID,
		VoteDefinitionID_2: voteDefinitionID,
		VoteDefinitionID_3: voteDefinitionID,
		VoteDefinitionID_4: voteDefinitionID,
		VoteDefinitionID_5: voteDefinitionID,
	})
	if err != nil {
		return nil, fmt.Errorf("get live vote submission stats: %w", err)
	}
	return &model.VoteSubmissionStats{
		EligibleCount:     row.EligibleCount,
		CastCount:         row.CastCount,
		BallotCount:       row.BallotCount,
		OpenBallotCount:   row.OpenBallotCount,
		SecretBallotCount: row.SecretBallotCount,
	}, nil
}

func (r *Repository) ensureVoteResultsReadable(ctx context.Context, voteDefinitionID int64) error {
	vd, err := r.Queries.GetVoteDefinitionByID(ctx, voteDefinitionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("vote definition not found")
		}
		return fmt.Errorf("load vote definition: %w", err)
	}
	if vd.State == model.VoteStateCounting {
		return fmt.Errorf("vote results unavailable while counting")
	}
	return nil
}

func validateVoteSelectionInput(ctx context.Context, qtx *client.Queries, vd client.VoteDefinition, optionIDs []int64) error {
	selectedCount := int64(len(optionIDs))
	if selectedCount < vd.MinSelections || selectedCount > vd.MaxSelections {
		return fmt.Errorf("invalid number of selections: got %d, expected between %d and %d", selectedCount, vd.MinSelections, vd.MaxSelections)
	}

	seenSelected := make(map[int64]struct{}, len(optionIDs))
	for _, optionID := range optionIDs {
		if _, exists := seenSelected[optionID]; exists {
			return fmt.Errorf("duplicate selected option id %d", optionID)
		}
		seenSelected[optionID] = struct{}{}
	}

	allowedOptionIDs, err := qtx.ListVoteOptionIDsForVoteDefinition(ctx, vd.ID)
	if err != nil {
		return fmt.Errorf("list vote option ids: %w", err)
	}
	allowed := make(map[int64]struct{}, len(allowedOptionIDs))
	for _, optionID := range allowedOptionIDs {
		allowed[optionID] = struct{}{}
	}
	for _, optionID := range optionIDs {
		if _, ok := allowed[optionID]; !ok {
			return fmt.Errorf("option %d does not belong to vote definition %d", optionID, vd.ID)
		}
	}
	return nil
}

func ptrToNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func parseNullRFC3339(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func voteDefinitionFromClient(v *client.VoteDefinition) *model.VoteDefinition {
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339, v.UpdatedAt)
	return &model.VoteDefinition{
		ID:            v.ID,
		MeetingID:     v.MeetingID,
		AgendaPointID: v.AgendaPointID,
		Name:          v.Name,
		Visibility:    v.Visibility,
		State:         v.State,
		MinSelections: v.MinSelections,
		MaxSelections: v.MaxSelections,
		OpenedAt:      parseNullRFC3339(v.OpenedAt),
		ClosedAt:      parseNullRFC3339(v.ClosedAt),
		ArchivedAt:    parseNullRFC3339(v.ArchivedAt),
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func voteOptionFromClient(v *client.VoteOption) *model.VoteOption {
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	return &model.VoteOption{
		ID:               v.ID,
		VoteDefinitionID: v.VoteDefinitionID,
		Label:            v.Label,
		Position:         v.Position,
		CreatedAt:        createdAt,
	}
}

func eligibleVoterFromClient(v *client.EligibleVoter) *model.EligibleVoter {
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	return &model.EligibleVoter{
		VoteDefinitionID: v.VoteDefinitionID,
		MeetingID:        v.MeetingID,
		AttendeeID:       v.AttendeeID,
		CreatedAt:        createdAt,
	}
}

func voteCastFromClient(v *client.VoteCast) *model.VoteCast {
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	return &model.VoteCast{
		ID:               v.ID,
		VoteDefinitionID: v.VoteDefinitionID,
		MeetingID:        v.MeetingID,
		AttendeeID:       v.AttendeeID,
		Source:           v.Source,
		CreatedAt:        createdAt,
	}
}

func voteBallotFromClient(v *client.VoteBallot) *model.VoteBallot {
	createdAt, _ := time.Parse(time.RFC3339, v.CreatedAt)
	result := &model.VoteBallot{
		ID:                v.ID,
		VoteDefinitionID:  v.VoteDefinitionID,
		CastID:            nullInt64ToPtr(v.CastID),
		AttendeeID:        nullInt64ToPtr(v.AttendeeID),
		ReceiptToken:      v.ReceiptToken,
		CommitmentCipher:  nil,
		CommitmentVersion: nullInt64ToPtr(v.CommitmentVersion),
		CreatedAt:         createdAt,
	}
	if v.EncryptedCommitment != nil {
		commitment := append([]byte(nil), v.EncryptedCommitment...)
		result.EncryptedCommitment = &commitment
	}
	if v.CommitmentCipher.Valid {
		cipher := v.CommitmentCipher.String
		result.CommitmentCipher = &cipher
	}
	return result
}
