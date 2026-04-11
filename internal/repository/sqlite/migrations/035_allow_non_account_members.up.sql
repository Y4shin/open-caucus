-- Allow committee members without a sitewide account.
-- Members can now be identified by email instead of account_id.
-- invite_secret stores a personalized token for email-only member login.

PRAGMA foreign_keys = OFF;

CREATE TABLE users_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id     INTEGER REFERENCES accounts(id) ON DELETE CASCADE,
    committee_id   INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    email          TEXT,
    full_name      TEXT    NOT NULL,
    role           TEXT    NOT NULL CHECK (role IN ('chairperson', 'member')),
    quoted         BOOLEAN NOT NULL DEFAULT FALSE,
    invite_secret  TEXT,
    created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, account_id),
    UNIQUE (committee_id, email)
);

INSERT INTO users_new (id, account_id, committee_id, full_name, role, quoted, created_at, updated_at)
SELECT u.id, u.account_id, u.committee_id, COALESCE(a.full_name, ''), u.role, u.quoted, u.created_at, u.updated_at
FROM users u
LEFT JOIN accounts a ON u.account_id = a.id;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

PRAGMA foreign_keys = ON;
