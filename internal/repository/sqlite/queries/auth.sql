-- name: GetAccountByUsername :one
-- Retrieves sitewide account for login authentication
SELECT * FROM accounts WHERE username = ?;

-- name: GetPasswordCredentialByAccountID :one
-- Retrieves the password credential for a given account
SELECT * FROM password_credentials WHERE account_id = ?;

-- name: GetUserMembershipByAccountAndCommittee :one
-- Retrieves the committee membership row for an account+committee combination,
-- including the username from the accounts table.
SELECT u.id, u.account_id, u.committee_id, u.full_name, u.role, u.quoted,
       u.created_at, u.updated_at, a.username
FROM users u
JOIN accounts a ON u.account_id = a.id
JOIN committees c ON u.committee_id = c.id
WHERE a.username = ? AND c.slug = ?;

-- name: GetCommitteeBySlug :one
-- Retrieves committee by slug
SELECT * FROM committees WHERE slug = ?;

-- name: GetUserByID :one
-- Retrieves membership row by ID (for session restoration), including username from accounts
SELECT u.id, u.account_id, u.committee_id, u.full_name, u.role, u.quoted,
       u.created_at, u.updated_at, a.username
FROM users u
JOIN accounts a ON u.account_id = a.id
WHERE u.id = ?;
