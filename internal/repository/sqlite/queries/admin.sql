-- Committee Management

-- name: ListAllCommittees :many
SELECT * FROM committees ORDER BY name ASC LIMIT ? OFFSET ?;

-- name: CountAllCommittees :one
SELECT COUNT(*) FROM committees;

-- name: CreateCommitteeWithSlug :one
INSERT INTO committees (name, slug, created_at, updated_at)
VALUES (?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: DeleteCommitteeBySlug :exec
DELETE FROM committees WHERE slug = ?;

-- User / Membership Management

-- name: ListUsersInCommittee :many
SELECT u.id, u.account_id, u.committee_id, a.full_name, u.role, u.quoted,
       u.created_at, u.updated_at, a.username
FROM users u
JOIN accounts a ON u.account_id = a.id
JOIN committees c ON u.committee_id = c.id
WHERE c.slug = ?
ORDER BY a.username ASC LIMIT ? OFFSET ?;

-- name: CountUsersInCommittee :one
SELECT COUNT(*) FROM users u
JOIN committees c ON u.committee_id = c.id
WHERE c.slug = ?;

-- name: CreateAccount :one
-- Creates a new sitewide account. Caller must also create a credential row.
INSERT INTO accounts (username, full_name, auth_method, created_at, updated_at)
VALUES (?, ?, 'password', datetime('now'), datetime('now'))
RETURNING *;

-- name: CreatePasswordCredential :one
-- Creates a password credential for an account.
INSERT INTO password_credentials (account_id, auth_method, password_hash, created_at, updated_at)
VALUES (?, 'password', ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: UpsertPasswordCredential :exec
-- Inserts or updates the password credential for an account.
INSERT INTO password_credentials (account_id, auth_method, password_hash, created_at, updated_at)
VALUES (?, 'password', ?, datetime('now'), datetime('now'))
ON CONFLICT (account_id) DO UPDATE
    SET password_hash = excluded.password_hash,
        updated_at    = datetime('now');

-- name: CreateMembership :one
-- Creates a committee membership row for an existing account.
INSERT INTO users (account_id, committee_id, role, quoted, created_at, updated_at)
VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: SetAccountIsAdmin :exec
UPDATE accounts SET is_admin = ?, updated_at = datetime('now') WHERE id = ?;


-- name: UpdateMembership :exec
UPDATE users
SET quoted = ?, role = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteUserByID :exec
DELETE FROM users WHERE id = ?;

-- name: GetCommitteeIDBySlug :one
SELECT id FROM committees WHERE slug = ?;

-- name: CountAllAccounts :one
SELECT COUNT(*) FROM accounts;

-- name: ListAllAccounts :many
SELECT * FROM accounts ORDER BY username ASC LIMIT ? OFFSET ?;

-- name: ListUnassignedAccountsForCommittee :many
SELECT a.*
FROM accounts a
LEFT JOIN users u
    ON u.account_id = a.id
   AND u.committee_id = ?
WHERE u.id IS NULL
ORDER BY a.username ASC;
