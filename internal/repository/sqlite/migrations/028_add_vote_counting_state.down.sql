PRAGMA foreign_keys = OFF;

CREATE TABLE vote_definitions_old (
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

INSERT INTO vote_definitions_old (
    id, meeting_id, agenda_point_id, motion_id, name, visibility, state,
    min_selections, max_selections, opened_at, closed_at, archived_at, created_at, updated_at
)
SELECT
    id,
    meeting_id,
    agenda_point_id,
    motion_id,
    name,
    visibility,
    CASE
        WHEN state = 'counting' THEN 'closed'
        ELSE state
    END,
    min_selections,
    max_selections,
    opened_at,
    closed_at,
    archived_at,
    created_at,
    updated_at
FROM vote_definitions;

DROP TABLE vote_definitions;
ALTER TABLE vote_definitions_old RENAME TO vote_definitions;

CREATE UNIQUE INDEX vote_definitions_id_meeting_unique
ON vote_definitions(id, meeting_id);

PRAGMA foreign_keys = ON;
