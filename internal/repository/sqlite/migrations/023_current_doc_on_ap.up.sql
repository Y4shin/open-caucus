ALTER TABLE agenda_points ADD COLUMN current_attachment_id INTEGER REFERENCES agenda_attachments(id) ON DELETE SET NULL;

ALTER TABLE agenda_points ADD COLUMN current_motion_id INTEGER REFERENCES motions(id) ON DELETE SET NULL;
