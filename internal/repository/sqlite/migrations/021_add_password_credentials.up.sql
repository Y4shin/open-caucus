-- Extract password credentials into a dedicated table and add auth_method to accounts.
--
-- accounts gains an auth_method column with UNIQUE(id, auth_method), enabling
-- compound foreign keys from credential tables. password_credentials uses
-- auth_method CHECK('password') and FOREIGN KEY(account_id, auth_method)
-- REFERENCES accounts(id, auth_method), so the DB enforces that a password
-- credential can only reference an account whose auth_method is also 'password'.
-- Future methods (oauth, ldap, …) follow the same pattern with their own tables.

PRAGMA foreign_keys = OFF;

-- Save password hashes to a temp table before dropping old accounts.
CREATE TABLE password_hashes_temp AS
SELECT id AS account_id, password_hash FROM accounts;

-- Rebuild accounts: add auth_method, drop password_hash, add UNIQUE(id, auth_method)
CREATE TABLE accounts_new (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    username    TEXT    NOT NULL UNIQUE,
    auth_method TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method IN ('password')),
    is_admin    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (id, auth_method)
);

INSERT INTO accounts_new (id, username, auth_method, is_admin, created_at, updated_at)
SELECT id, username, 'password', is_admin, created_at, updated_at FROM accounts;

-- Replace accounts (now has auth_method + UNIQUE(id, auth_method))
DROP TABLE accounts;
ALTER TABLE accounts_new RENAME TO accounts;

-- Create password_credentials now that accounts has the right schema for the compound FK
CREATE TABLE password_credentials (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL UNIQUE,
    auth_method   TEXT    NOT NULL DEFAULT 'password' CHECK(auth_method = 'password'),
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (account_id, auth_method) REFERENCES accounts(id, auth_method)
);

-- Migrate password hashes from temp table
INSERT INTO password_credentials (account_id, auth_method, password_hash, created_at, updated_at)
SELECT account_id, 'password', password_hash, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
FROM password_hashes_temp;

DROP TABLE password_hashes_temp;

PRAGMA foreign_keys = ON;
