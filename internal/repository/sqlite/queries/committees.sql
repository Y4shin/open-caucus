-- name: GetCommittee :one
SELECT * FROM committees WHERE id = ?;

-- name: ListCommittees :many
SELECT * FROM committees ORDER BY name;

-- name: CreateCommittee :one
INSERT INTO committees (name) VALUES (?) RETURNING *;
