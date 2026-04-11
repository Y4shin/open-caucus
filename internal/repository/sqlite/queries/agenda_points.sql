-- name: GetMaxAgendaPointPosition :one
SELECT COALESCE(MAX(position), 0) FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL;

-- name: GetMaxSubAgendaPointPosition :one
SELECT COALESCE(MAX(position), 0) FROM agenda_points
WHERE meeting_id = ? AND parent_id = ?;

-- name: CreateAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, NULL, ?, ?)
RETURNING *;

-- name: CreateSubAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListAgendaPointsForMeeting :many
SELECT * FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL
ORDER BY position ASC;

-- name: ListSubAgendaPointsForMeeting :many
SELECT * FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NOT NULL
ORDER BY parent_id ASC, position ASC;

-- name: ListSubAgendaPointsForParent :many
SELECT * FROM agenda_points
WHERE meeting_id = ? AND parent_id = ?
ORDER BY position ASC;

-- name: GetAgendaPointByID :one
SELECT * FROM agenda_points WHERE id = ?;

-- name: UpdateAgendaPointTitle :exec
UPDATE agenda_points SET title = ? WHERE id = ?;

-- name: DeleteAgendaPoint :exec
DELETE FROM agenda_points WHERE id = ?;

-- name: SetAgendaPointPosition :exec
UPDATE agenda_points
SET position = ?
WHERE id = ?;

-- name: UpdateAgendaPointStructure :exec
UPDATE agenda_points
SET parent_id = ?, position = ?, title = ?
WHERE id = ? AND meeting_id = ?;

-- name: BumpAgendaPointPositionsForMeeting :exec
UPDATE agenda_points
SET position = position + 1000000
WHERE meeting_id = ?;

-- name: SetCurrentAgendaPoint :exec
UPDATE meetings SET current_agenda_point_id = ? WHERE id = ?;

-- name: SetCurrentSpeaker :exec
UPDATE agenda_points SET current_speaker_id = ? WHERE id = ?;

-- name: SetAgendaPointQuotationOrder :exec
UPDATE agenda_points SET quotation_order = ? WHERE id = ?;

-- name: SetAgendaPointModerator :exec
UPDATE agenda_points SET moderator_id = ? WHERE id = ?;

-- name: SetCurrentAttachment :exec
UPDATE agenda_points
SET current_attachment_id = ?
WHERE id = ?;

-- name: ClearCurrentDocument :exec
UPDATE agenda_points
SET current_attachment_id = NULL
WHERE id = ?;

-- name: SetAgendaPointEnteredAt :exec
UPDATE agenda_points SET entered_at = ? WHERE id = ?;

-- name: SetAgendaPointLeftAt :exec
UPDATE agenda_points SET left_at = ? WHERE id = ?;
