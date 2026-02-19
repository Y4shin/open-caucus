# Phase 1 — Database Schema & Code Generation ✅ DONE

## Goal

Add all new columns via a migration, update SQL queries to include those columns, and regenerate the SQLC database client. After this phase the project must compile (Phase 2 will wire up the Go-layer changes).

---

## 1. Migration

Create two files:

### `internal/repository/sqlite/migrations/017_add_quotation_and_order.up.sql`

```sql
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
```

### `internal/repository/sqlite/migrations/017_add_quotation_and_order.down.sql`

```sql
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
```

---

## 2. SQL Query Changes

### `internal/repository/sqlite/queries/speakers.sql`

**Replace `AddSpeaker`** — add `gender_quoted` and `first_speaker` as INSERT columns and include all new columns in RETURNING:

```sql
-- name: AddSpeaker :one
INSERT INTO speakers_list (agenda_point_id, attendee_id, type, gender_quoted, first_speaker)
VALUES (?, ?, ?, ?, ?)
RETURNING id, agenda_point_id, attendee_id, type, status,
          requested_at, start_of_speech, duration,
          gender_quoted, first_speaker, priority, order_position;
```

**Replace `ListSpeakersForAgendaPoint`** — add new columns and change ORDER BY:

```sql
-- name: ListSpeakersForAgendaPoint :many
SELECT sl.id, sl.agenda_point_id, sl.attendee_id, sl.type, sl.status,
       sl.requested_at, sl.start_of_speech, sl.duration,
       sl.gender_quoted, sl.first_speaker, sl.priority, sl.order_position,
       a.full_name AS attendee_full_name
FROM speakers_list sl
JOIN attendees a ON a.id = sl.attendee_id
WHERE sl.agenda_point_id = ?
ORDER BY
    CASE sl.status
        WHEN 'SPEAKING' THEN 0
        WHEN 'WAITING'  THEN sl.order_position + 1
        WHEN 'DONE'     THEN 1000000
        ELSE                 1000001
    END ASC,
    sl.requested_at ASC;
```

**Replace `GetSpeakerEntryByID`** — add new columns:

```sql
-- name: GetSpeakerEntryByID :one
SELECT id, agenda_point_id, attendee_id, type, status,
       requested_at, start_of_speech, duration,
       gender_quoted, first_speaker, priority, order_position
FROM speakers_list WHERE id = ?;
```

**Add new queries** at the end of the file:

```sql
-- name: HasAttendeeSpokenOnAgendaPoint :one
SELECT EXISTS(
    SELECT 1 FROM speakers_list
    WHERE agenda_point_id = ? AND attendee_id = ? AND status IN ('SPEAKING', 'DONE')
);

-- name: GetWaitingSpeakersForAgendaPoint :many
SELECT id, agenda_point_id, attendee_id, type, status,
       requested_at, start_of_speech, duration,
       gender_quoted, first_speaker, priority, order_position
FROM speakers_list
WHERE agenda_point_id = ? AND status = 'WAITING'
ORDER BY order_position ASC;

-- name: SetSpeakerPriority :exec
UPDATE speakers_list SET priority = ? WHERE id = ?;

-- name: SetSpeakerOrderPosition :exec
UPDATE speakers_list SET order_position = ? WHERE id = ?;
```

---

### `internal/repository/sqlite/queries/meetings.sql`

Add at the end of the file:

```sql
-- name: SetMeetingGenderQuotation :exec
UPDATE meetings SET gender_quotation_enabled = ? WHERE id = ?;

-- name: SetMeetingFirstSpeakerQuotation :exec
UPDATE meetings SET first_speaker_quotation_enabled = ? WHERE id = ?;

-- name: SetMeetingModerator :exec
UPDATE meetings SET moderator_id = ? WHERE id = ?;
```

> `GetMeetingByID` uses `SELECT *` so it automatically picks up the new columns — no change needed.

---

### `internal/repository/sqlite/queries/agenda_points.sql`

**Replace `ListAgendaPointsForMeeting`** — add new columns:

```sql
-- name: ListAgendaPointsForMeeting :many
SELECT id, meeting_id, parent_id, position, title, protocol,
       created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id
FROM agenda_points
WHERE meeting_id = ? AND parent_id IS NULL
ORDER BY position ASC;
```

**Replace `GetAgendaPointByID`** — add new columns:

```sql
-- name: GetAgendaPointByID :one
SELECT id, meeting_id, parent_id, position, title, protocol,
       created_at, updated_at, current_speaker_id,
       gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id
FROM agenda_points WHERE id = ?;
```

**Replace `CreateAgendaPoint` RETURNING clause** — add new columns:

```sql
-- name: CreateAgendaPoint :one
INSERT INTO agenda_points (meeting_id, parent_id, position, title)
VALUES (?, NULL, ?, ?)
RETURNING id, meeting_id, parent_id, position, title, protocol,
          created_at, updated_at, current_speaker_id,
          gender_quotation_enabled, first_speaker_quotation_enabled, moderator_id;
```

**Add new queries** at the end of the file:

```sql
-- name: SetAgendaPointGenderQuotation :exec
UPDATE agenda_points SET gender_quotation_enabled = ? WHERE id = ?;

-- name: SetAgendaPointFirstSpeakerQuotation :exec
UPDATE agenda_points SET first_speaker_quotation_enabled = ? WHERE id = ?;

-- name: SetAgendaPointModerator :exec
UPDATE agenda_points SET moderator_id = ? WHERE id = ?;
```

> The nullable boolean parameters (`gender_quotation_enabled`, `first_speaker_quotation_enabled`) on agenda_points accept `NULL` to clear the override (inherit from meeting). SQLC will generate `sql.NullBool` for the nullable `BOOLEAN` column. Pass `sql.NullBool{Valid: false}` to set NULL.

---

## 3. Regenerate Database Client

```bash
task generate:db
```

SQLC reads the schema from the migration files and the queries above, and regenerates `internal/repository/sqlite/client/`.

---

## 4. Verification

After codegen, the project will have compile errors in `internal/repository/sqlite/repository.go` and `internal/handlers/manage.go` because the generated types now have new fields and the `AddSpeaker` query signature changed. That is expected — Phase 2 fixes those. For now, just confirm codegen itself succeeds (no SQLC errors):

```bash
task generate:db   # must exit 0
```

You can optionally run `go build ./...` to see the expected compile errors and confirm they are only related to the changed `AddSpeaker` signature and new struct fields — not unexpected issues.
