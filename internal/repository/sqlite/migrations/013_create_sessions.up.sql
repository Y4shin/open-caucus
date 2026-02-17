CREATE TABLE sessions (
    session_id   TEXT PRIMARY KEY,
    session_type TEXT NOT NULL CHECK (session_type IN ('user', 'attendee')),

    -- For user sessions
    user_id        INTEGER REFERENCES users(id) ON DELETE CASCADE,
    committee_slug TEXT,

    -- For attendee sessions
    attendee_id INTEGER REFERENCES attendees(id) ON DELETE CASCADE,
    meeting_id  INTEGER REFERENCES meetings(id) ON DELETE CASCADE,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL,

    -- Ensure exactly one of user_id or attendee_id is set
    CHECK (
        (session_type = 'user' AND user_id IS NOT NULL AND attendee_id IS NULL) OR
        (session_type = 'attendee' AND attendee_id IS NOT NULL AND user_id IS NULL)
    )
);

-- Index for cleanup of expired sessions
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
