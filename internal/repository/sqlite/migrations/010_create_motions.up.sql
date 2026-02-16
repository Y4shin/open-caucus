CREATE TABLE motions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    agenda_point_id INTEGER NOT NULL REFERENCES agenda_points(id) ON DELETE CASCADE,
    blob_id         INTEGER NOT NULL REFERENCES binary_blobs(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL,
    votes_for       INTEGER,
    votes_against   INTEGER,
    votes_abstained INTEGER,
    votes_eligible  INTEGER,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (
        (votes_for IS NULL AND votes_against IS NULL AND votes_abstained IS NULL AND votes_eligible IS NULL)
        OR (votes_for IS NOT NULL AND votes_against IS NOT NULL AND votes_abstained IS NOT NULL AND votes_eligible IS NOT NULL)
    )
);
