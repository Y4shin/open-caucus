ALTER TABLE agenda_points ADD COLUMN protocol TEXT NOT NULL DEFAULT '';

ALTER TABLE meetings ADD COLUMN protocol_writer_id INTEGER REFERENCES attendees(id) ON DELETE SET NULL;
