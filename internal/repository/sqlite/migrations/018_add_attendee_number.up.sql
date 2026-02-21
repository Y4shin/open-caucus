ALTER TABLE attendees ADD COLUMN attendee_number INTEGER;

WITH numbered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (PARTITION BY meeting_id ORDER BY created_at, id) AS rn
    FROM attendees
)
UPDATE attendees
SET attendee_number = (
    SELECT rn
    FROM numbered
    WHERE numbered.id = attendees.id
);

CREATE UNIQUE INDEX attendees_meeting_attendee_number_unique
ON attendees(meeting_id, attendee_number);
