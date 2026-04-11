-- Revert: make account_id NOT NULL again, drop email and invite_secret.
-- Email-only members (account_id IS NULL) are deleted since they cannot exist
-- in the old schema.

PRAGMA foreign_keys = OFF;

DELETE FROM users WHERE account_id IS NULL;

CREATE TABLE users_old (
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

INSERT INTO users_old (id, account_id, committee_id, full_name, role, quoted, created_at, updated_at)
SELECT id, account_id, committee_id, full_name, role, quoted, created_at, updated_at FROM users;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

PRAGMA foreign_keys = ON;
