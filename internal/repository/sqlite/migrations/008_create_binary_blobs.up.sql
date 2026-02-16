CREATE TABLE binary_blobs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    filename     TEXT    NOT NULL,
    content_type TEXT    NOT NULL,
    size_bytes   INTEGER NOT NULL,
    storage_path TEXT    NOT NULL,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
