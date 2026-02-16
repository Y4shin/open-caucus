CREATE TABLE attendees (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    meeting_id INTEGER NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id    INTEGER          REFERENCES users(id) ON DELETE SET NULL,
    full_name  TEXT    NOT NULL,
    gender     TEXT    NOT NULL CHECK (gender IN ('m', 'f', 'd')),
    secret     TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
