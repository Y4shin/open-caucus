PRAGMA foreign_keys = OFF;

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
SELECT
    u.id,
    u.account_id,
    u.committee_id,
    COALESCE(a.full_name, a.username),
    u.role,
    u.quoted,
    u.created_at,
    u.updated_at
FROM users u
JOIN accounts a ON a.id = u.account_id;

DROP TABLE users;
ALTER TABLE users_old RENAME TO users;

ALTER TABLE accounts DROP COLUMN full_name;

PRAGMA foreign_keys = ON;
