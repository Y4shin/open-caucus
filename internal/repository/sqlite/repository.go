package sqlite

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite"

	"github.com/Y4shin/conference-tool/internal/repository/sqlite/client"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Repository wraps the database connection and generated sqlc client.
type Repository struct {
	DB      *sql.DB
	Queries *client.Queries
}

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
