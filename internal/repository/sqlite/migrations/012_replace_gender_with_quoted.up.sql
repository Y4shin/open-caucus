-- Replace gender with quoted flag in users table
ALTER TABLE users DROP COLUMN gender;
ALTER TABLE users ADD COLUMN quoted BOOLEAN NOT NULL DEFAULT FALSE;

-- Replace gender with quoted flag in attendees table
ALTER TABLE attendees DROP COLUMN gender;
ALTER TABLE attendees ADD COLUMN quoted BOOLEAN NOT NULL DEFAULT FALSE;
