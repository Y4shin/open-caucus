CREATE TABLE speakers_list (
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
