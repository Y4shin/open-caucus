-- name: AddSpeaker :one
INSERT INTO speakers_list (agenda_point_id, attendee_id, type, gender_quoted, first_speaker)
VALUES (?, ?, ?, ?, ?)
RETURNING id, agenda_point_id, attendee_id, type, status,
          requested_at, start_of_speech, duration,
          gender_quoted, first_speaker, priority, order_position;

-- name: ListSpeakersForAgendaPoint :many
SELECT sl.id, sl.agenda_point_id, sl.attendee_id, sl.type, sl.status,
       sl.requested_at, sl.start_of_speech, sl.duration,
       sl.gender_quoted, sl.first_speaker, sl.priority, sl.order_position,
       a.full_name AS attendee_full_name
FROM speakers_list sl
JOIN attendees a ON a.id = sl.attendee_id
WHERE sl.agenda_point_id = ?
ORDER BY
    CASE sl.status
        WHEN 'SPEAKING' THEN 0
        WHEN 'WAITING'  THEN sl.order_position + 1
        WHEN 'DONE'     THEN 1000000
        ELSE                 1000001
    END ASC,
    sl.requested_at ASC;

-- name: GetSpeakerEntryByID :one
SELECT id, agenda_point_id, attendee_id, type, status,
       requested_at, start_of_speech, duration,
       gender_quoted, first_speaker, priority, order_position
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

-- name: HasAttendeeSpokenOnAgendaPoint :one
SELECT EXISTS(
    SELECT 1 FROM speakers_list
    WHERE agenda_point_id = ? AND attendee_id = ? AND status IN ('SPEAKING', 'DONE')
);

-- name: GetWaitingSpeakersForAgendaPoint :many
SELECT id, agenda_point_id, attendee_id, type, status,
       requested_at, start_of_speech, duration,
       gender_quoted, first_speaker, priority, order_position
FROM speakers_list
WHERE agenda_point_id = ? AND status = 'WAITING'
ORDER BY order_position ASC;

-- name: SetSpeakerPriority :exec
UPDATE speakers_list SET priority = ? WHERE id = ?;

-- name: SetSpeakerOrderPosition :exec
UPDATE speakers_list SET order_position = ? WHERE id = ?;
