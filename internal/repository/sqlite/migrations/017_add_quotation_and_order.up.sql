-- speakers_list: quotation snapshots, ordering, manual priority
ALTER TABLE speakers_list ADD COLUMN gender_quoted  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE speakers_list ADD COLUMN first_speaker  BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE speakers_list ADD COLUMN priority       BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE speakers_list ADD COLUMN order_position INTEGER NOT NULL DEFAULT 0;

-- meetings: quotation toggles and optional moderator
ALTER TABLE meetings ADD COLUMN gender_quotation_enabled        BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE meetings ADD COLUMN first_speaker_quotation_enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE meetings ADD COLUMN moderator_id INTEGER REFERENCES attendees(id) ON DELETE SET NULL;

-- agenda_points: per-point overrides (NULL = inherit from meeting)
ALTER TABLE agenda_points ADD COLUMN gender_quotation_enabled        BOOLEAN;
ALTER TABLE agenda_points ADD COLUMN first_speaker_quotation_enabled BOOLEAN;
ALTER TABLE agenda_points ADD COLUMN moderator_id INTEGER REFERENCES attendees(id) ON DELETE SET NULL;
