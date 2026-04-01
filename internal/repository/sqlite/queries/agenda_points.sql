-- name: GetMaxAgendaPointPosition :one
SELECT COALESCE(MAX(position), 0) FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL;

-- name: GetMaxSubAgendaPointPosition :one
SELECT COALESCE(MAX(position), 0) FROM agenda_points
WHERE meeting_id = ? AND parent_id = ?;

-- name: CreateAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, NULL, ?, ?)
RETURNING id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
          gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
          current_attachment_id;

-- name: CreateSubAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, ?, ?, ?)
RETURNING id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
          gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
          current_attachment_id;

-- name: ListAgendaPointsForMeeting :many
SELECT id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
       current_attachment_id
FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL
ORDER BY position ASC;

-- name: ListSubAgendaPointsForMeeting :many
SELECT id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
       current_attachment_id
FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NOT NULL
ORDER BY parent_id ASC, position ASC;

-- name: ListSubAgendaPointsForParent :many
SELECT id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
       current_attachment_id
FROM agenda_points
WHERE meeting_id = ? AND parent_id = ?
ORDER BY position ASC;

-- name: GetAgendaPointByID :one
SELECT id, meeting_id, parent_id, position, title, created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id,
       current_attachment_id
FROM agenda_points WHERE id = ?;

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

-- name: SetAgendaPointGenderQuotation :exec
UPDATE agenda_points SET gender_quotation_enabled = ? WHERE id = ?;

-- name: SetAgendaPointFirstSpeakerQuotation :exec
UPDATE agenda_points SET first_speaker_quotation_enabled = ? WHERE id = ?;

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
