-- Introduce sitewide accounts table and rework users as committee memberships.
--
-- accounts holds authentication credentials (username + password_hash) once per
-- sitewide identity. The users table is rebuilt to reference accounts via
-- account_id, dropping the per-committee username and password_hash columns.
-- SQLite does not support DROP COLUMN on columns involved in constraints, so
-- the table-rebuild pattern is used.

PRAGMA foreign_keys = OFF;

CREATE TABLE accounts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE users_new (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    committee_id INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    full_name    TEXT    NOT NULL,
    role         TEXT    NOT NULL CHECK (role IN ('chairperson', 'member')),
    quoted       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, account_id)
);

-- No data migration: this migration targets empty databases only.

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

PRAGMA foreign_keys = ON;
