CREATE TABLE agenda_attachments (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    blob_id         INTEGER NOT NULL REFERENCES binary_blobs(id) ON DELETE CASCADE,
    label           TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (agenda_point_id, blob_id)
);
