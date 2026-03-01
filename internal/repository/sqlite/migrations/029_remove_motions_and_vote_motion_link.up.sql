PRAGMA foreign_keys = OFF;

CREATE TABLE vote_definitions_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id      INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    agenda_point_id INTEGER NOT NULL,
    name            TEXT    NOT NULL,
    visibility      TEXT    NOT NULL CHECK (visibility IN ('open', 'secret')),
    state           TEXT    NOT NULL DEFAULT 'draft' CHECK (state IN ('draft', 'open', 'counting', 'closed', 'archived')),
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
    FOREIGN KEY (agenda_point_id, meeting_id) REFERENCES agenda_points(id, meeting_id) ON DELETE CASCADE
);

INSERT INTO vote_definitions_new (
    id, meeting_id, agenda_point_id, name, visibility, state,
    min_selections, max_selections, opened_at, closed_at, archived_at, created_at, updated_at
)
SELECT
    id, meeting_id, agenda_point_id, name, visibility, state,
    min_selections, max_selections, opened_at, closed_at, archived_at, created_at, updated_at
FROM vote_definitions;

DROP TABLE vote_definitions;
ALTER TABLE vote_definitions_new RENAME TO vote_definitions;

CREATE UNIQUE INDEX vote_definitions_id_meeting_unique
ON vote_definitions(id, meeting_id);

CREATE TABLE agenda_points_new (
    id                             INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id                     INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    parent_id                      INTEGER REFERENCES agenda_points_new(id) ON DELETE SET NULL,
    position                       INTEGER NOT NULL,
    title                          TEXT    NOT NULL,
    created_at                     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at                     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    current_speaker_id             INTEGER REFERENCES speakers_list(id) ON DELETE SET NULL,
    gender_quotation_enabled       BOOLEAN,
    first_speaker_quotation_enabled BOOLEAN,
    moderator_id                   INTEGER REFERENCES attendees(id) ON DELETE SET NULL,
    current_attachment_id          INTEGER REFERENCES agenda_attachments(id) ON DELETE SET NULL
);

INSERT INTO agenda_points_new (
    id, meeting_id, parent_id, position, title, created_at, updated_at,
    current_speaker_id, gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id, current_attachment_id
)
SELECT
    id, meeting_id, parent_id, position, title, created_at, updated_at,
    current_speaker_id, gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id, current_attachment_id
FROM agenda_points;

DROP TABLE agenda_points;
ALTER TABLE agenda_points_new RENAME TO agenda_points;

CREATE UNIQUE INDEX agenda_points_id_meeting_unique
ON agenda_points(id, meeting_id);

DROP INDEX IF EXISTS motions_id_agenda_point_unique;
DROP TABLE IF EXISTS motions;

PRAGMA foreign_keys = ON;
