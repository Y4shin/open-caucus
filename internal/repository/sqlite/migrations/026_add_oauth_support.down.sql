PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS oauth_managed_memberships;
DROP TABLE IF EXISTS oauth_committee_group_rules;
DROP TABLE IF EXISTS oauth_identities;

CREATE TABLE accounts_old (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL UNIQUE,
    auth_method TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method IN ('password')),
    is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    full_name   TEXT,
    UNIQUE (id, auth_method)
);

INSERT INTO accounts_old (id, username, auth_method, is_admin, created_at, updated_at, full_name)
SELECT id, username, auth_method, is_admin, created_at, updated_at, full_name
FROM accounts;

CREATE TABLE password_credentials_old (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL UNIQUE,
    auth_method   TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method = 'password'),
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (account_id, auth_method) REFERENCES accounts_old(id, auth_method)
);

INSERT INTO password_credentials_old (id, account_id, auth_method, password_hash, created_at, updated_at)
SELECT id, account_id, auth_method, password_hash, created_at, updated_at
FROM password_credentials;

DROP TABLE password_credentials;
DROP TABLE accounts;
ALTER TABLE accounts_old RENAME TO accounts;
ALTER TABLE password_credentials_old RENAME TO password_credentials;

PRAGMA foreign_keys = ON;

