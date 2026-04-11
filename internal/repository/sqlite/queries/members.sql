-- name: CreateEmailMember :one
INSERT INTO users (committee_id, email, full_name, role, quoted, invite_secret, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: GetUserByCommitteeAndEmail :one
SELECT u.id, u.account_id, u.committee_id, u.email,
       u.full_name, u.role, u.quoted, u.invite_secret,
       u.created_at, u.updated_at,
       '' AS username,
       CASE WHEN om.user_id IS NULL THEN 0 ELSE 1 END AS oauth_managed
FROM users u
LEFT JOIN oauth_managed_memberships om ON om.user_id = u.id
JOIN committees c ON u.committee_id = c.id
WHERE c.slug = ? AND u.email = ?;

-- name: GetUserByInviteSecret :one
SELECT u.id, u.account_id, u.committee_id, u.email,
       u.full_name, u.role, u.quoted, u.invite_secret,
       u.created_at, u.updated_at,
       COALESCE(a.username, '') AS username,
       c.slug AS committee_slug,
       CASE WHEN om.user_id IS NULL THEN 0 ELSE 1 END AS oauth_managed
FROM users u
LEFT JOIN accounts a ON u.account_id = a.id
JOIN committees c ON u.committee_id = c.id
LEFT JOIN oauth_managed_memberships om ON om.user_id = u.id
WHERE u.invite_secret = ?;

-- name: SetInviteSecret :exec
UPDATE users SET invite_secret = ?, updated_at = datetime('now') WHERE id = ?;

-- name: ListAllMembersForCommittee :many
SELECT u.id, u.account_id, u.committee_id, u.email,
       COALESCE(a.full_name, u.full_name) AS full_name,
       u.role, u.quoted, u.invite_secret,
       u.created_at, u.updated_at,
       COALESCE(a.username, '') AS username,
       CASE WHEN om.user_id IS NULL THEN 0 ELSE 1 END AS oauth_managed
FROM users u
LEFT JOIN accounts a ON u.account_id = a.id
JOIN committees c ON u.committee_id = c.id
LEFT JOIN oauth_managed_memberships om ON om.user_id = u.id
WHERE c.slug = ?
ORDER BY COALESCE(a.username, u.email, u.full_name) ASC;
