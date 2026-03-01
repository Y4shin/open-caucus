PRAGMA foreign_keys = OFF;

CREATE TABLE motions_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    blob_id         INTEGER NOT NULL REFERENCES binary_blobs(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT INTO motions_new (id, agenda_point_id, blob_id, title, created_at, updated_at)
SELECT id, agenda_point_id, blob_id, title, created_at, updated_at
FROM motions;

DROP TABLE motions;
ALTER TABLE motions_new RENAME TO motions;

CREATE UNIQUE INDEX motions_id_agenda_point_unique
ON motions(id, agenda_point_id);

CREATE UNIQUE INDEX agenda_points_id_meeting_unique
ON agenda_points(id, meeting_id);

CREATE UNIQUE INDEX attendees_id_meeting_unique
ON attendees(id, meeting_id);

CREATE TABLE vote_definitions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id      INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    agenda_point_id INTEGER NOT NULL,
    motion_id       INTEGER,
    name            TEXT    NOT NULL,
    visibility      TEXT    NOT NULL CHECK (visibility IN ('open', 'secret')),
    state           TEXT    NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'open', 'closed', 'archived')),
    min_selections  INTEGER NOT NULL,
    max_selections  INTEGER NOT NULL,
    opened_at       TEXT,
    closed_at       TEXT,
    archived_at     TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (min_selections >= 0),
    CHECK (max_selections >= 1),
    CHECK (min_selections <= max_selections),
    FOREIGN KEY (agenda_point_id, meeting_id) REFERENCES agenda_points(id, meeting_id) ON DELETE CASCADE,
    FOREIGN KEY (motion_id, agenda_point_id) REFERENCES motions(id, agenda_point_id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX vote_definitions_id_meeting_unique
ON vote_definitions(id, meeting_id);

CREATE TABLE vote_options (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    vote_definition_id INTEGER NOT NULL REFERENCES vote_definitions(id) ON DELETE CASCADE,
    label              TEXT    NOT NULL,
    position           INTEGER NOT NULL,
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (trim(label) <> ''),
    UNIQUE (vote_definition_id, position)
);

CREATE UNIQUE INDEX vote_options_id_vote_definition_unique
ON vote_options(id, vote_definition_id);

CREATE TABLE eligible_voters (
    vote_definition_id INTEGER NOT NULL,
    meeting_id         INTEGER NOT NULL,
    attendee_id        INTEGER NOT NULL,
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (vote_definition_id, attendee_id),
    FOREIGN KEY (vote_definition_id, meeting_id) REFERENCES vote_definitions(id, meeting_id) ON DELETE CASCADE,
    FOREIGN KEY (attendee_id, meeting_id) REFERENCES attendees(id, meeting_id) ON DELETE CASCADE
);

CREATE TABLE vote_casts (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    vote_definition_id INTEGER NOT NULL,
    meeting_id         INTEGER NOT NULL,
    attendee_id        INTEGER NOT NULL,
    source             TEXT    NOT NULL CHECK (source IN ('self_submission', 'manual_submission')),
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (vote_definition_id, attendee_id),
    FOREIGN KEY (vote_definition_id, meeting_id) REFERENCES vote_definitions(id, meeting_id) ON DELETE CASCADE,
    FOREIGN KEY (attendee_id, meeting_id) REFERENCES attendees(id, meeting_id) ON DELETE CASCADE,
    FOREIGN KEY (vote_definition_id, attendee_id) REFERENCES eligible_voters(vote_definition_id, attendee_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX vote_casts_id_vote_definition_unique
ON vote_casts(id, vote_definition_id);

CREATE UNIQUE INDEX vote_casts_id_vote_definition_attendee_unique
ON vote_casts(id, vote_definition_id, attendee_id);

CREATE TABLE vote_ballots (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    vote_definition_id   INTEGER NOT NULL REFERENCES vote_definitions(id) ON DELETE CASCADE,
    cast_id              INTEGER,
    attendee_id          INTEGER,
    receipt_token        TEXT    NOT NULL,
    encrypted_commitment BLOB,
    commitment_cipher    TEXT,
    commitment_version   INTEGER,
    created_at           TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (vote_definition_id, receipt_token),
    UNIQUE (cast_id),
    UNIQUE (vote_definition_id, attendee_id),
    FOREIGN KEY (cast_id, vote_definition_id, attendee_id) REFERENCES vote_casts(id, vote_definition_id, attendee_id) ON DELETE CASCADE,
    FOREIGN KEY (vote_definition_id, attendee_id) REFERENCES eligible_voters(vote_definition_id, attendee_id) ON DELETE CASCADE,
    CHECK (
        (
            attendee_id IS NOT NULL
            AND cast_id IS NOT NULL
            AND encrypted_commitment IS NULL
            AND commitment_cipher IS NULL
            AND commitment_version IS NULL
        )
        OR
        (
            attendee_id IS NULL
            AND cast_id IS NULL
            AND encrypted_commitment IS NOT NULL
            AND commitment_cipher IS NOT NULL
            AND commitment_version IS NOT NULL
        )
    )
);

CREATE UNIQUE INDEX vote_ballots_id_vote_definition_unique
ON vote_ballots(id, vote_definition_id);

CREATE TABLE vote_ballot_selections (
    ballot_id           INTEGER NOT NULL,
    vote_definition_id  INTEGER NOT NULL,
    option_id           INTEGER NOT NULL,
    created_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (ballot_id, option_id),
    FOREIGN KEY (ballot_id, vote_definition_id) REFERENCES vote_ballots(id, vote_definition_id) ON DELETE CASCADE,
    FOREIGN KEY (option_id, vote_definition_id) REFERENCES vote_options(id, vote_definition_id) ON DELETE CASCADE
);

PRAGMA foreign_keys = ON;
