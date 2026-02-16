ALTER TABLE agenda_points DROP COLUMN current_speaker_id;

ALTER TABLE meetings DROP COLUMN current_agenda_point_id;

ALTER TABLE committees DROP COLUMN current_meeting_id;
