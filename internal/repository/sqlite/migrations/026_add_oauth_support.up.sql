PRAGMA foreign_keys = OFF;

CREATE TABLE accounts_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL UNIQUE,
    auth_method TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method IN ('password', 'oauth')),
    is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    full_name   TEXT,
    UNIQUE (id, auth_method)
);

INSERT INTO accounts_new (id, username, auth_method, is_admin, created_at, updated_at, full_name)
SELECT id, username, auth_method, is_admin, created_at, updated_at, full_name
FROM accounts;

CREATE TABLE password_credentials_new (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL UNIQUE,
    auth_method   TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method = 'password'),
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (account_id, auth_method) REFERENCES accounts_new(id, auth_method)
);

INSERT INTO password_credentials_new (id, account_id, auth_method, password_hash, created_at, updated_at)
SELECT id, account_id, auth_method, password_hash, created_at, updated_at
FROM password_credentials;

DROP TABLE password_credentials;
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;
ALTER TABLE password_credentials_new RENAME TO password_credentials;

CREATE TABLE oauth_identities (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    issuer      TEXT    NOT NULL,
    subject     TEXT    NOT NULL,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    username    TEXT,
    full_name   TEXT,
    email       TEXT,
    groups_json TEXT,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (issuer, subject),
    UNIQUE (account_id)
);

CREATE TABLE oauth_committee_group_rules (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    committee_id INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    group_name  TEXT    NOT NULL,
    role        TEXT    NOT NULL CHECK(role IN ('chairperson', 'member')),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, group_name)
);

CREATE TABLE oauth_managed_memberships (
    user_id         INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_synced_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

PRAGMA foreign_keys = ON;

