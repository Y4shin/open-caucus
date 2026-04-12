-- name: InsertSentEmail :exec
INSERT INTO sent_emails (message_id, recipient, committee_id, meeting_id, subject)
VALUES (?, ?, ?, ?, ?);

-- name: ListSentEmailsByRecipientAndCommittee :many
SELECT message_id FROM sent_emails
WHERE recipient = ? AND committee_id = ?
ORDER BY created_at ASC;
