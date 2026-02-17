-- Restore gender column in attendees table (default to 'd' for diverse/unknown)
ALTER TABLE attendees DROP COLUMN quoted;
ALTER TABLE attendees ADD COLUMN gender TEXT NOT NULL DEFAULT 'd' CHECK (gender IN ('m', 'f', 'd'));

-- Restore gender column in users table (default to 'd' for diverse/unknown)
ALTER TABLE users DROP COLUMN quoted;
ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'd' CHECK (gender IN ('m', 'f', 'd'));