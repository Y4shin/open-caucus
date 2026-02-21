-- name: CreateAttendee :one
INSERT INTO attendees (meeting_id, user_id, full_name, secret, attendee_number)
VALUES (
    ?, ?, ?, ?,
    COALESCE((SELECT MAX(a.attendee_number) + 1 FROM attendees a WHERE a.meeting_id = ?), 1)
)
RETURNING *;

-- name: GetAttendeeByUserIDAndMeetingID :one
SELECT * FROM attendees WHERE user_id = ? AND meeting_id = ?;

-- name: GetAttendeeByID :one
SELECT * FROM attendees WHERE id = ?;

-- name: GetAttendeeByMeetingIDAndSecret :one
SELECT * FROM attendees WHERE meeting_id = ? AND secret = ?;

-- name: ListAttendeesForMeeting :many
SELECT * FROM attendees WHERE meeting_id = ? ORDER BY attendee_number ASC, created_at ASC;

-- name: DeleteAttendee :exec
DELETE FROM attendees WHERE id = ?;

-- name: SetAttendeeIsChair :exec
UPDATE attendees SET is_chair = ? WHERE id = ?;
