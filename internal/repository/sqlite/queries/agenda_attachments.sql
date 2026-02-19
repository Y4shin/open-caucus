-- name: CreateAgendaAttachment :one
INSERT INTO agenda_attachments (agenda_point_id, blob_id, label)
VALUES (?, ?, ?)
RETURNING id, agenda_point_id, blob_id, label, created_at;

-- name: GetAgendaAttachmentByID :one
SELECT id, agenda_point_id, blob_id, label, created_at
FROM agenda_attachments
WHERE id = ?;

-- name: ListAttachmentsForAgendaPoint :many
SELECT id, agenda_point_id, blob_id, label, created_at
FROM agenda_attachments
WHERE agenda_point_id = ?
ORDER BY created_at ASC;

-- name: DeleteAgendaAttachment :exec
DELETE FROM agenda_attachments WHERE id = ?;
