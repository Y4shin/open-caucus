-- Reverse migration 021: merge password_credentials back into accounts and drop auth_method.

PRAGMA foreign_keys = OFF;

CREATE TABLE accounts_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL DEFAULT '',
    is_admin      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT INTO accounts_new (id, username, password_hash, is_admin, created_at, updated_at)
SELECT a.id, a.username, COALESCE(pc.password_hash, ''), a.is_admin, a.created_at, a.updated_at
FROM accounts a
LEFT JOIN password_credentials pc ON pc.account_id = a.id;

DROP TABLE password_credentials;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;

PRAGMA foreign_keys = ON;
