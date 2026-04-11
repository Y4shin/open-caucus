-- name: ListMeetingsForCommittee :many
SELECT * FROM meetings
WHERE committee_id = (SELECT id FROM committees WHERE slug = ?)
ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountMeetingsForCommittee :one
SELECT COUNT(*) FROM meetings
WHERE committee_id = (SELECT id FROM committees WHERE slug = ?);

-- name: CreateMeeting :exec
INSERT INTO meetings (committee_id, name, description, secret, signup_open, start_at, end_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetMeetingByID :one
SELECT * FROM meetings WHERE id = ?;

-- name: DeleteMeeting :exec
DELETE FROM meetings WHERE id = ?;

-- name: SetActiveMeeting :exec
UPDATE committees SET current_meeting_id = ? WHERE slug = ?;

-- name: SetMeetingSignupOpen :exec
UPDATE meetings SET signup_open = ? WHERE id = ?;

-- name: SetMeetingSignupOpenWithVersion :one
UPDATE meetings SET signup_open = ?, version = version + 1 WHERE id = ? RETURNING version;

-- name: SetMeetingGenderQuotation :exec
UPDATE meetings SET gender_quotation_enabled = ? WHERE id = ?;

-- name: SetMeetingFirstSpeakerQuotation :exec
UPDATE meetings SET first_speaker_quotation_enabled = ? WHERE id = ?;

-- name: SetMeetingModerator :exec
UPDATE meetings SET moderator_id = ? WHERE id = ?;

-- name: SetMeetingDatetime :exec
UPDATE meetings SET start_at = ?, end_at = ? WHERE id = ?;
