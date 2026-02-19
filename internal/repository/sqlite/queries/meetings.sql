-- name: ListMeetingsForCommittee :many
SELECT * FROM meetings
WHERE committee_id = (SELECT id FROM committees WHERE slug = ?)
ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountMeetingsForCommittee :one
SELECT COUNT(*) FROM meetings
WHERE committee_id = (SELECT id FROM committees WHERE slug = ?);

-- name: CreateMeeting :exec
INSERT INTO meetings (committee_id, name, description, secret, signup_open)
VALUES (?, ?, ?, ?, ?);

-- name: GetMeetingByID :one
SELECT * FROM meetings WHERE id = ?;

-- name: DeleteMeeting :exec
DELETE FROM meetings WHERE id = ?;

-- name: SetActiveMeeting :exec
UPDATE committees SET current_meeting_id = ? WHERE slug = ?;

-- name: SetMeetingSignupOpen :exec
UPDATE meetings SET signup_open = ? WHERE id = ?;

-- name: SetMeetingProtocolWriter :exec
UPDATE meetings SET protocol_writer_id = ? WHERE id = ?;
