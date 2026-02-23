-- Reverse migration 019: restore users to per-committee credentials shape
-- and drop the accounts table.

PRAGMA foreign_keys = OFF;

CREATE TABLE users_old (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    committee_id  INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    username      TEXT    NOT NULL,
    password_hash TEXT    NOT NULL,
    full_name     TEXT    NOT NULL,
    role          TEXT    NOT NULL CHECK (role IN ('chairperson', 'member')),
    quoted        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, username)
);

-- No data migration: this migration targets empty databases only.

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

DROP TABLE IF EXISTS accounts;

PRAGMA foreign_keys = ON;
