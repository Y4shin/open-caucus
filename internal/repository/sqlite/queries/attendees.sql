-- name: CreateAttendee :one
INSERT INTO attendees (meeting_id, user_id, full_name, secret)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetAttendeeByUserIDAndMeetingID :one
SELECT * FROM attendees WHERE user_id = ? AND meeting_id = ?;

-- name: GetAttendeeByID :one
SELECT * FROM attendees WHERE id = ?;

-- name: GetAttendeeByMeetingIDAndSecret :one
SELECT * FROM attendees WHERE meeting_id = ? AND secret = ?;

-- name: ListAttendeesForMeeting :many
SELECT * FROM attendees WHERE meeting_id = ? ORDER BY created_at ASC;

-- name: DeleteAttendee :exec
DELETE FROM attendees WHERE id = ?;

-- name: SetAttendeeIsChair :exec
UPDATE attendees SET is_chair = ? WHERE id = ?;
