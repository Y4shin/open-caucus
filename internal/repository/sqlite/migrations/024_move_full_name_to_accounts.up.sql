PRAGMA foreign_keys = OFF;

ALTER TABLE accounts ADD COLUMN full_name TEXT;

UPDATE accounts
SET full_name = COALESCE(
    (
        SELECT u.full_name
        FROM users u
        WHERE u.account_id = accounts.id
        ORDER BY u.id
        LIMIT 1
    ),
    username
)
WHERE full_name IS NULL OR full_name = '';

CREATE TABLE users_new (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    committee_id INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    role         TEXT    NOT NULL CHECK (role IN ('chairperson', 'member')),
    quoted       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, account_id)
);

INSERT INTO users_new (id, account_id, committee_id, role, quoted, created_at, updated_at)
SELECT id, account_id, committee_id, role, quoted, created_at, updated_at
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

PRAGMA foreign_keys = ON;
