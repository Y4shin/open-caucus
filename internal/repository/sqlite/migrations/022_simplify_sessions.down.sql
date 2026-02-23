-- Reverse migration 022: restore the denormalized 12-column sessions table
-- with user/attendee/admin session types (as introduced by migration 020).

PRAGMA foreign_keys = OFF;

DROP TABLE sessions;

CREATE TABLE sessions (
    session_id   TEXT PRIMARY KEY,
    session_type TEXT NOT NULL CHECK (session_type IN ('user', 'attendee', 'admin')),

    -- For user sessions
    user_id        INTEGER REFERENCES users(id) ON DELETE CASCADE,
    committee_slug TEXT,

    -- For attendee sessions
    attendee_id INTEGER REFERENCES attendees(id) ON DELETE CASCADE,
    meeting_id  INTEGER REFERENCES meetings(id) ON DELETE CASCADE,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL,

    -- Cached identity fields
    username  TEXT,
    role      TEXT,
    full_name TEXT,
    is_chair  INTEGER,
    quoted    INTEGER,

    CHECK (
        (session_type = 'user'     AND user_id IS NOT NULL AND attendee_id IS NULL) OR
        (session_type = 'attendee' AND attendee_id IS NOT NULL AND user_id IS NULL) OR
        (session_type = 'admin'    AND user_id IS NULL AND attendee_id IS NULL)
    )
);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

PRAGMA foreign_keys = ON;
