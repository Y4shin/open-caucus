-- Remove chair flag from attendees
ALTER TABLE attendees DROP COLUMN is_chair;

-- Remove protocol writer from meetings
ALTER TABLE meetings DROP COLUMN protocol_writer_id;
