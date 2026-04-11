-- Add ordered quotation configuration, replacing the two boolean columns.
-- The JSON array encodes both order and enabled state: present = enabled, position = priority.

ALTER TABLE meetings ADD COLUMN quotation_order TEXT NOT NULL DEFAULT '["gender","first_speaker"]';

-- Migrate existing boolean settings to JSON array.
UPDATE meetings SET quotation_order = CASE
    WHEN gender_quotation_enabled = 1 AND first_speaker_quotation_enabled = 1 THEN '["gender","first_speaker"]'
    WHEN gender_quotation_enabled = 1 AND first_speaker_quotation_enabled = 0 THEN '["gender"]'
    WHEN gender_quotation_enabled = 0 AND first_speaker_quotation_enabled = 1 THEN '["first_speaker"]'
    ELSE '[]'
END;

ALTER TABLE meetings DROP COLUMN gender_quotation_enabled;
ALTER TABLE meetings DROP COLUMN first_speaker_quotation_enabled;

-- Same for agenda points (nullable = inherit from meeting).
ALTER TABLE agenda_points ADD COLUMN quotation_order TEXT;

-- Migrate existing agenda point overrides.
UPDATE agenda_points SET quotation_order = CASE
    WHEN gender_quotation_enabled IS NOT NULL AND first_speaker_quotation_enabled IS NOT NULL THEN
        CASE
            WHEN gender_quotation_enabled = 1 AND first_speaker_quotation_enabled = 1 THEN '["gender","first_speaker"]'
            WHEN gender_quotation_enabled = 1 AND first_speaker_quotation_enabled = 0 THEN '["gender"]'
            WHEN gender_quotation_enabled = 0 AND first_speaker_quotation_enabled = 1 THEN '["first_speaker"]'
            ELSE '[]'
        END
    WHEN gender_quotation_enabled IS NOT NULL THEN
        CASE WHEN gender_quotation_enabled = 1 THEN '["gender"]' ELSE '[]' END
    WHEN first_speaker_quotation_enabled IS NOT NULL THEN
        CASE WHEN first_speaker_quotation_enabled = 1 THEN '["first_speaker"]' ELSE '[]' END
    ELSE NULL
END
WHERE gender_quotation_enabled IS NOT NULL OR first_speaker_quotation_enabled IS NOT NULL;

ALTER TABLE agenda_points DROP COLUMN gender_quotation_enabled;
ALTER TABLE agenda_points DROP COLUMN first_speaker_quotation_enabled;
