ALTER TABLE committees ADD COLUMN current_meeting_id INTEGER REFERENCES meetings(id) ON DELETE SET NULL;

ALTER TABLE meetings ADD COLUMN current_agenda_point_id INTEGER REFERENCES agenda_points(id) ON DELETE SET NULL;

ALTER TABLE agenda_points ADD COLUMN current_speaker_id INTEGER REFERENCES speakers_list(id) ON DELETE SET NULL;
