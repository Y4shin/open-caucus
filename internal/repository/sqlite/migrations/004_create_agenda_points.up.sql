CREATE TABLE agenda_points (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    parent_id  INTEGER          REFERENCES agenda_points(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    title      TEXT    NOT NULL,
    protocol   TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (meeting_id, parent_id, position)
);
