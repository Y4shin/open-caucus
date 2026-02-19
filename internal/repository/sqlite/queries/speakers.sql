-- name: AddSpeaker :one
INSERT INTO speakers_list (agenda_point_id, attendee_id, type)
VALUES (?, ?, ?)
RETURNING id, agenda_point_id, attendee_id, type, status, requested_at, start_of_speech, duration;

-- name: ListSpeakersForAgendaPoint :many
SELECT sl.id, sl.agenda_point_id, sl.attendee_id, sl.type, sl.status,
       sl.requested_at, sl.start_of_speech, sl.duration,
       a.full_name AS attendee_full_name
FROM speakers_list sl
JOIN attendees a ON a.id = sl.attendee_id
WHERE sl.agenda_point_id = ?
ORDER BY sl.requested_at ASC;

-- name: GetSpeakerEntryByID :one
SELECT id, agenda_point_id, attendee_id, type, status, requested_at, start_of_speech, duration
FROM speakers_list WHERE id = ?;

-- name: SetSpeakerSpeaking :exec
UPDATE speakers_list
SET status = 'SPEAKING', start_of_speech = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ?;

-- name: SetSpeakerDone :exec
UPDATE speakers_list SET status = 'DONE', duration = ? WHERE id = ?;

-- name: SetSpeakerWithdrawn :exec
UPDATE speakers_list SET status = 'WITHDRAWN' WHERE id = ?;

-- name: DeleteSpeaker :exec
DELETE FROM speakers_list WHERE id = ?;
