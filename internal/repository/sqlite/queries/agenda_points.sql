-- name: GetMaxAgendaPointPosition :one
SELECT COALESCE(MAX(position), 0) FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL;

-- name: CreateAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, NULL, ?, ?)
RETURNING id, meeting_id, parent_id, position, title, protocol, created_at, updated_at, current_speaker_id;

-- name: ListAgendaPointsForMeeting :many
SELECT id, meeting_id, parent_id, position, title, protocol, created_at, updated_at, current_speaker_id
FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL
ORDER BY position ASC;

-- name: GetAgendaPointByID :one
SELECT id, meeting_id, parent_id, position, title, protocol, created_at, updated_at, current_speaker_id
FROM agenda_points WHERE id = ?;

-- name: DeleteAgendaPoint :exec
DELETE FROM agenda_points WHERE id = ?;

-- name: SetCurrentAgendaPoint :exec
UPDATE meetings SET current_agenda_point_id = ? WHERE id = ?;

-- name: SetCurrentSpeaker :exec
UPDATE agenda_points SET current_speaker_id = ? WHERE id = ?;

-- name: UpdateAgendaPointProtocol :exec
UPDATE agenda_points SET protocol = ? WHERE id = ?;
