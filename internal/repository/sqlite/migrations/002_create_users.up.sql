CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    committee_id  INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    username      TEXT    NOT NULL,
    password_hash TEXT    NOT NULL,
    full_name     TEXT    NOT NULL,
    gender        TEXT    NOT NULL CHECK (gender IN ('m', 'f', 'd')),
    role          TEXT    NOT NULL CHECK (role IN ('chairperson', 'member')),
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (committee_id, username)
);
