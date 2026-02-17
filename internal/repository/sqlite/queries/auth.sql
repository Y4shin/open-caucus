-- name: GetUserByCommitteeAndUsername :one
-- Retrieves user for login authentication
SELECT u.*
FROM users u
JOIN committees c ON u.committee_id = c.id
WHERE c.slug = ? AND u.username = ?;

-- name: GetCommitteeBySlug :one
-- Retrieves committee by slug
SELECT * FROM committees WHERE slug = ?;

-- name: GetUserByID :one
-- Retrieves user by ID (for session restoration if needed)
SELECT * FROM users WHERE id = ?;
