ALTER TABLE speakers_list DROP COLUMN gender_quoted;
ALTER TABLE speakers_list DROP COLUMN first_speaker;
ALTER TABLE speakers_list DROP COLUMN priority;
ALTER TABLE speakers_list DROP COLUMN order_position;

ALTER TABLE meetings DROP COLUMN gender_quotation_enabled;
ALTER TABLE meetings DROP COLUMN first_speaker_quotation_enabled;
ALTER TABLE meetings DROP COLUMN moderator_id;

ALTER TABLE agenda_points DROP COLUMN gender_quotation_enabled;
ALTER TABLE agenda_points DROP COLUMN first_speaker_quotation_enabled;
ALTER TABLE agenda_points DROP COLUMN moderator_id;
