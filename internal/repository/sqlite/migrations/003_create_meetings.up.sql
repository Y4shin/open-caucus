CREATE TABLE meetings (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    committee_id INTEGER NOT NULL REFERENCES committees(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    secret       TEXT    NOT NULL,
    signup_open  BOOLEAN NOT NULL,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
