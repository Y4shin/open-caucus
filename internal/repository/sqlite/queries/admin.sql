-- Committee Management

-- name: ListAllCommittees :many
SELECT * FROM committees ORDER BY name ASC;

-- name: CreateCommitteeWithSlug :one
INSERT INTO committees (name, slug, created_at, updated_at)
VALUES (?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: DeleteCommitteeBySlug :exec
DELETE FROM committees WHERE slug = ?;

-- User Management

-- name: ListUsersInCommittee :many
SELECT u.* FROM users u
JOIN committees c ON u.committee_id = c.id
WHERE c.slug = ?
ORDER BY u.username ASC;

-- name: CreateUser :one
INSERT INTO users (
    committee_id, username, password_hash, full_name,
    quoted, role, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))
RETURNING *;

-- name: UpdateUser :exec
UPDATE users
SET password_hash = ?, full_name = ?, quoted = ?, role = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: DeleteUserByID :exec
DELETE FROM users WHERE id = ?;

-- name: GetCommitteeIDBySlug :one
SELECT id FROM committees WHERE slug = ?;
