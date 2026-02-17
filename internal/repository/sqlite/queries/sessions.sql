-- name: CreateSession :exec
INSERT INTO sessions (
    session_id, session_type, user_id, committee_slug,
    attendee_id, meeting_id, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetSession :one
SELECT * FROM sessions WHERE session_id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE session_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < ?;
