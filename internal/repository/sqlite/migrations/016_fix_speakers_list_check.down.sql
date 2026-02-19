-- Restore original table with the (start_of_speech, duration) pair constraint.
CREATE TABLE speakers_list_old (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    attendee_id     INTEGER NOT NULL REFERENCES attendees(id) ON DELETE CASCADE,
    type            TEXT    NOT NULL CHECK (type IN ('regular', 'ropm')),
    status          TEXT    NOT NULL DEFAULT 'WAITING' CHECK (status IN ('WAITING', 'SPEAKING', 'DONE', 'WITHDRAWN')),
    requested_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    start_of_speech TEXT,
    duration        TEXT,
    CHECK (
        (start_of_speech IS NULL AND duration IS NULL)
        OR (start_of_speech IS NOT NULL AND duration IS NOT NULL)
    )
);

INSERT INTO speakers_list_old SELECT * FROM speakers_list WHERE start_of_speech IS NULL OR duration IS NOT NULL;
DROP TABLE speakers_list;
ALTER TABLE speakers_list_old RENAME TO speakers_list;
