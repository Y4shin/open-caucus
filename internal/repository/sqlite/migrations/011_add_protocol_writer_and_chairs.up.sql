-- Add protocol writer to meetings
ALTER TABLE meetings ADD COLUMN protocol_writer_id INTEGER REFERENCES attendees(id) ON DELETE SET NULL;

-- Add chair flag to attendees
ALTER TABLE attendees ADD COLUMN is_chair BOOLEAN NOT NULL DEFAULT FALSE;
