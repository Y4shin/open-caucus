-- name: CreateVoteDefinition :one
INSERT INTO vote_definitions (
    meeting_id, agenda_point_id, motion_id, name, visibility, state, min_selections, max_selections
)
VALUES (?, ?, ?, ?, ?, 'draft', ?, ?)
RETURNING *;

-- name: UpdateVoteDefinitionDraft :one
UPDATE vote_definitions
SET meeting_id = ?,
    agenda_point_id = ?,
    motion_id = ?,
    name = ?,
    visibility = ?,
    min_selections = ?,
    max_selections = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'draft'
RETURNING *;

-- name: SetVoteDefinitionOpen :one
UPDATE vote_definitions
SET state = 'open',
    opened_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'draft'
RETURNING *;

-- name: SetVoteDefinitionCountingFromOpen :one
UPDATE vote_definitions
SET state = 'counting',
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'open'
RETURNING *;

-- name: SetVoteDefinitionClosedFromOpen :one
UPDATE vote_definitions
SET state = 'closed',
    closed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'open'
RETURNING *;

-- name: SetVoteDefinitionClosedFromCounting :one
UPDATE vote_definitions
SET state = 'closed',
    closed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'counting'
RETURNING *;

-- name: SetVoteDefinitionArchived :one
UPDATE vote_definitions
SET state = 'archived',
    archived_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ? AND state = 'closed'
RETURNING *;

-- name: GetVoteDefinitionByID :one
SELECT * FROM vote_definitions WHERE id = ?;

-- name: ListVoteDefinitionsForAgendaPoint :many
SELECT *
FROM vote_definitions
WHERE agenda_point_id = ?
ORDER BY created_at ASC;

-- name: CountEligibleVotersForVoteDefinition :one
SELECT COUNT(*) FROM eligible_voters WHERE vote_definition_id = ?;

-- name: DeleteVoteOptionsForVoteDefinition :exec
DELETE FROM vote_options WHERE vote_definition_id = ?;

-- name: CreateVoteOption :one
INSERT INTO vote_options (vote_definition_id, label, position)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListVoteOptionsForVoteDefinition :many
SELECT *
FROM vote_options
WHERE vote_definition_id = ?
ORDER BY position ASC, id ASC;

-- name: InsertEligibleVoter :exec
INSERT INTO eligible_voters (vote_definition_id, meeting_id, attendee_id)
VALUES (?, ?, ?);

-- name: ListEligibleVotersForVoteDefinition :many
SELECT *
FROM eligible_voters
WHERE vote_definition_id = ?
ORDER BY attendee_id ASC;

-- name: CreateVoteCast :one
INSERT INTO vote_casts (vote_definition_id, meeting_id, attendee_id, source)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetVoteCastByVoteAndAttendee :one
SELECT *
FROM vote_casts
WHERE vote_definition_id = ? AND attendee_id = ?;

-- name: ListVoteCastsForVoteDefinition :many
SELECT *
FROM vote_casts
WHERE vote_definition_id = ?
ORDER BY created_at ASC;

-- name: CreateOpenVoteBallot :one
INSERT INTO vote_ballots (vote_definition_id, cast_id, attendee_id, receipt_token)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: CreateSecretVoteBallot :one
INSERT INTO vote_ballots (
    vote_definition_id, cast_id, attendee_id, receipt_token, encrypted_commitment, commitment_cipher, commitment_version
)
VALUES (?, NULL, NULL, ?, ?, ?, ?)
RETURNING *;

-- name: CreateVoteBallotSelection :exec
INSERT INTO vote_ballot_selections (ballot_id, vote_definition_id, option_id)
VALUES (?, ?, ?);

-- name: CountSecretVoteBallots :one
SELECT COUNT(*)
FROM vote_ballots
WHERE vote_definition_id = ? AND attendee_id IS NULL;

-- name: CountVoteCastsForVoteDefinition :one
SELECT COUNT(*)
FROM vote_casts
WHERE vote_definition_id = ?;

-- name: ListVoteOptionIDsForVoteDefinition :many
SELECT id
FROM vote_options
WHERE vote_definition_id = ?;

-- name: GetOpenVoteVerificationRows :many
SELECT
    vd.id            AS vote_definition_id,
    vd.name          AS vote_name,
    a.id             AS attendee_id,
    a.attendee_number,
    vb.receipt_token,
    vo.id            AS option_id,
    vo.label         AS option_label
FROM vote_ballots vb
JOIN vote_definitions vd ON vd.id = vb.vote_definition_id
JOIN attendees a ON a.id = vb.attendee_id
LEFT JOIN vote_ballot_selections vbs ON vbs.ballot_id = vb.id AND vbs.vote_definition_id = vb.vote_definition_id
LEFT JOIN vote_options vo ON vo.id = vbs.option_id AND vo.vote_definition_id = vb.vote_definition_id
WHERE vb.vote_definition_id = ? AND vb.receipt_token = ? AND vb.attendee_id IS NOT NULL
ORDER BY vo.position ASC, vo.id ASC;

-- name: GetSecretVoteVerification :one
SELECT
    vd.id                 AS vote_definition_id,
    vd.name               AS vote_name,
    vb.receipt_token,
    vb.encrypted_commitment,
    vb.commitment_cipher,
    vb.commitment_version
FROM vote_ballots vb
JOIN vote_definitions vd ON vd.id = vb.vote_definition_id
WHERE vb.vote_definition_id = ? AND vb.receipt_token = ? AND vb.attendee_id IS NULL
LIMIT 1;

-- name: GetVoteTallies :many
SELECT
    vo.id AS option_id,
    vo.label AS option_label,
    CAST(COALESCE(COUNT(vbs.option_id), 0) AS INTEGER) AS tally_count
FROM vote_options vo
LEFT JOIN vote_ballot_selections vbs ON vbs.option_id = vo.id AND vbs.vote_definition_id = vo.vote_definition_id
WHERE vo.vote_definition_id = ?
GROUP BY vo.id, vo.label, vo.position
ORDER BY vo.position ASC, vo.id ASC;

-- name: GetVoteSubmissionStats :one
SELECT
    (SELECT COUNT(*) FROM eligible_voters ev WHERE ev.vote_definition_id = ?) AS eligible_count,
    (SELECT COUNT(*) FROM vote_casts vc WHERE vc.vote_definition_id = ?) AS cast_count,
    (SELECT COUNT(*) FROM vote_ballots vb WHERE vb.vote_definition_id = ?) AS ballot_count,
    (SELECT COUNT(*) FROM vote_ballots vb WHERE vb.vote_definition_id = ? AND vb.attendee_id IS NOT NULL) AS open_ballot_count,
    (SELECT COUNT(*) FROM vote_ballots vb WHERE vb.vote_definition_id = ? AND vb.attendee_id IS NULL) AS secret_ballot_count;
