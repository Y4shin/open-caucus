CREATE TABLE sent_emails (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id   TEXT    NOT NULL,
    recipient    TEXT    NOT NULL,
    committee_id INTEGER REFERENCES committees(id) ON DELETE SET NULL,
    meeting_id   INTEGER REFERENCES meetings(id) ON DELETE SET NULL,
    subject      TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_sent_emails_recipient_committee ON sent_emails (recipient, committee_id);
