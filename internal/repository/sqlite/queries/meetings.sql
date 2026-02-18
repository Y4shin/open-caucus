-- name: ListMeetingsForCommittee :many
SELECT * FROM meetings
WHERE committee_id = (SELECT id FROM committees WHERE slug = ?)
ORDER BY created_at DESC;

-- name: CreateMeeting :exec
INSERT INTO meetings (committee_id, name, description, secret, signup_open)
VALUES (?, ?, ?, ?, ?);
