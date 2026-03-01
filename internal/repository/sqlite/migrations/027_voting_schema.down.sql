PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS vote_ballot_selections;
DROP TABLE IF EXISTS vote_ballots;
DROP TABLE IF EXISTS vote_casts;
DROP TABLE IF EXISTS eligible_voters;
DROP TABLE IF EXISTS vote_options;
DROP TABLE IF EXISTS vote_definitions;

DROP INDEX IF EXISTS vote_ballots_id_vote_definition_unique;
DROP INDEX IF EXISTS vote_casts_id_vote_definition_attendee_unique;
DROP INDEX IF EXISTS vote_casts_id_vote_definition_unique;
DROP INDEX IF EXISTS vote_options_id_vote_definition_unique;
DROP INDEX IF EXISTS vote_definitions_id_meeting_unique;

DROP INDEX IF EXISTS motions_id_agenda_point_unique;
DROP INDEX IF EXISTS agenda_points_id_meeting_unique;
DROP INDEX IF EXISTS attendees_id_meeting_unique;

CREATE TABLE motions_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    blob_id         INTEGER NOT NULL REFERENCES binary_blobs(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL,
    votes_for       INTEGER,
    votes_against   INTEGER,
    votes_abstained INTEGER,
    votes_eligible  INTEGER,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (
        (votes_for IS NULL AND votes_against IS NULL AND votes_abstained IS NULL AND votes_eligible IS NULL)
        OR (votes_for IS NOT NULL AND votes_against IS NOT NULL AND votes_abstained IS NOT NULL AND votes_eligible IS NOT NULL)
    )
);

INSERT INTO motions_old (
    id, agenda_point_id, blob_id, title,
    votes_for, votes_against, votes_abstained, votes_eligible,
    created_at, updated_at
)
SELECT
    id, agenda_point_id, blob_id, title,
    NULL, NULL, NULL, NULL,
    created_at, updated_at
FROM motions;

DROP TABLE motions;
ALTER TABLE motions_old RENAME TO motions;

PRAGMA foreign_keys = ON;
