-- name: CreateMotion :one
INSERT INTO motions (agenda_point_id, blob_id, title)
VALUES (?, ?, ?)
RETURNING id, agenda_point_id, blob_id, title, votes_for, votes_against, votes_abstained, votes_eligible, created_at, updated_at;

-- name: GetMotionByID :one
SELECT id, agenda_point_id, blob_id, title, votes_for, votes_against, votes_abstained, votes_eligible, created_at, updated_at
FROM motions WHERE id = ?;

-- name: ListMotionsForAgendaPoint :many
SELECT id, agenda_point_id, blob_id, title, votes_for, votes_against, votes_abstained, votes_eligible, created_at, updated_at
FROM motions WHERE agenda_point_id = ? ORDER BY created_at ASC;

-- name: DeleteMotion :exec
DELETE FROM motions WHERE id = ?;

-- name: SetMotionVotes :exec
UPDATE motions
SET votes_for = ?, votes_against = ?, votes_abstained = ?, votes_eligible = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ?;
