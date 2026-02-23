-- Simplify sessions table: remove denormalized identity fields, remove admin
-- session type, and rename user→account / attendee→guest.

PRAGMA foreign_keys = OFF;

DROP TABLE sessions;

CREATE TABLE sessions (
    session_id   TEXT PRIMARY KEY,
    session_type TEXT NOT NULL CHECK (session_type IN ('account', 'guest')),

    -- For account sessions (formerly 'user' sessions)
    account_id  INTEGER REFERENCES accounts(id) ON DELETE CASCADE,

    -- For guest sessions (formerly 'attendee' sessions)
    attendee_id INTEGER REFERENCES attendees(id) ON DELETE CASCADE,

    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at TEXT NOT NULL,

    CHECK (
        (session_type = 'account' AND account_id IS NOT NULL AND attendee_id IS NULL) OR
        (session_type = 'guest'   AND attendee_id IS NOT NULL AND account_id IS NULL)
    )
);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

PRAGMA foreign_keys = ON;
