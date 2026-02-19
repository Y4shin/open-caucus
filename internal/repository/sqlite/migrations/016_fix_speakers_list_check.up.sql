-- The original CHECK required start_of_speech and duration to be set together,
-- which prevented recording a start time without an end time.
-- Recreate the table without that constraint.
CREATE TABLE speakers_list_new (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    attendee_id     INTEGER NOT NULL REFERENCES attendees(id) ON DELETE CASCADE,
    type            TEXT    NOT NULL CHECK (type IN ('regular', 'ropm')),
    status          TEXT    NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING', 'SPEAKING', 'DONE', 'WITHDRAWN')),
    requested_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    start_of_speech TEXT,
    duration        TEXT
);

INSERT INTO speakers_list_new SELECT * FROM speakers_list;
DROP TABLE speakers_list;
ALTER TABLE speakers_list_new RENAME TO speakers_list;
