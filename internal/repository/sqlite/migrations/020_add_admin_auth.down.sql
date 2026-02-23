-- Reverse migration 020: remove is_admin from accounts and rebuild sessions
-- to drop the 'admin' session type.

PRAGMA foreign_keys = OFF;

CREATE TABLE sessions_old (
    session_id   TEXT PRIMARY KEY,
    session_type TEXT NOT NULL CHECK (session_type IN ('user', 'attendee')),

    user_id        INTEGER REFERENCES users(id) ON DELETE CASCADE,
    committee_slug TEXT,

    attendee_id INTEGER REFERENCES attendees(id) ON DELETE CASCADE,
    meeting_id  INTEGER REFERENCES meetings(id) ON DELETE CASCADE,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL,

    username  TEXT,
    role      TEXT,
    full_name TEXT,
    is_chair  INTEGER,
    quoted    INTEGER,

    CHECK (
        (session_type = 'user'     AND user_id IS NOT NULL AND attendee_id IS NULL) OR
        (session_type = 'attendee' AND attendee_id IS NOT NULL AND user_id IS NULL)
    )
);

CREATE INDEX idx_sessions_old_expires_at ON sessions_old(expires_at);

-- No data migration: targeting empty databases only.

DROP TABLE sessions;
ALTER TABLE sessions_old RENAME TO sessions;

PRAGMA foreign_keys = ON;

ALTER TABLE accounts DROP COLUMN is_admin;
