-- Restore boolean quotation columns from JSON array.

ALTER TABLE meetings ADD COLUMN gender_quotation_enabled BOOLEAN NOT NULL DEFAULT 1;
ALTER TABLE meetings ADD COLUMN first_speaker_quotation_enabled BOOLEAN NOT NULL DEFAULT 1;

UPDATE meetings SET
    gender_quotation_enabled = CASE WHEN instr(quotation_order, '"gender"') > 0 THEN 1 ELSE 0 END,
    first_speaker_quotation_enabled = CASE WHEN instr(quotation_order, '"first_speaker"') > 0 THEN 1 ELSE 0 END;

ALTER TABLE meetings DROP COLUMN quotation_order;

ALTER TABLE agenda_points ADD COLUMN gender_quotation_enabled BOOLEAN;
ALTER TABLE agenda_points ADD COLUMN first_speaker_quotation_enabled BOOLEAN;

UPDATE agenda_points SET
    gender_quotation_enabled = CASE WHEN quotation_order IS NOT NULL AND instr(quotation_order, '"gender"') > 0 THEN 1 WHEN quotation_order IS NOT NULL THEN 0 ELSE NULL END,
    first_speaker_quotation_enabled = CASE WHEN quotation_order IS NOT NULL AND instr(quotation_order, '"first_speaker"') > 0 THEN 1 WHEN quotation_order IS NOT NULL THEN 0 ELSE NULL END
WHERE quotation_order IS NOT NULL;

ALTER TABLE agenda_points DROP COLUMN quotation_order;
