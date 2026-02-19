# Phase 2 — Models, Repository Interface & Implementation ✅ DONE

## Goal

Update Go model structs to reflect new columns, extend the repository interface with new methods, implement those methods in the SQLite repository, and fix the one changed call site in the handlers. After this phase the project must build and all tests must pass.

---

## 1. Model Updates

### `internal/repository/model/speaker_entry.go`

Add four fields:

```go
type SpeakerEntry struct {
    ID            int64
    AgendaPointID int64
    AttendeeID    int64
    AttendeeName  string
    Type          string  // "regular" or "ropm"
    Status        string  // "WAITING", "SPEAKING", "DONE", "WITHDRAWN"
    GenderQuoted  bool    // snapshot: was gender quotation applied at request time?
    FirstSpeaker  bool    // snapshot: was this the attendee's first time on this AP?
    Priority      bool    // manually promoted by chairperson
    OrderPosition int64   // computed ordering index (meaningful only for WAITING)
}
```

### `internal/repository/model/meeting.go`

Add three fields:

```go
type Meeting struct {
    ID                           int64
    Name                         string
    Description                  string
    SignupOpen                   bool
    CurrentAgendaPointID        *int64
    ProtocolWriterID             *int64
    GenderQuotationEnabled       bool   // default true
    FirstSpeakerQuotationEnabled bool   // default true
    ModeratorID                  *int64 // nil if not set
    CreatedAt                    time.Time
}
```

### `internal/repository/model/agenda_point.go`

Add three fields (nullable booleans use `*bool` — nil means "inherit from meeting"):

```go
type AgendaPoint struct {
    ID                           int64
    MeetingID                    int64
    Position                     int64
    Title                        string
    Protocol                     string
    CurrentSpeakerID            *int64
    GenderQuotationEnabled       *bool  // nil = inherit from meeting
    FirstSpeakerQuotationEnabled *bool  // nil = inherit from meeting
    ModeratorID                  *int64 // nil if not set
}
```

---

## 2. Repository Interface (`internal/repository/repository.go`)

### Update existing method signature

```go
// Changed: added genderQuoted and firstSpeaker parameters
AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error)
```

### New speakers methods

```go
// Returns true if the attendee has any SPEAKING or DONE entry for this agenda point.
HasAttendeeSpokenOnAgendaPoint(ctx context.Context, agendaPointID, attendeeID int64) (bool, error)

// Toggle manual priority on a speakers-list entry.
SetSpeakerPriority(ctx context.Context, id int64, priority bool) error

// Recompute and persist order_position for all WAITING speakers on the given agenda point.
// Sort key: priority DESC, gender_quoted DESC, first_speaker DESC, requested_at ASC.
// Assigns positions 1, 2, 3, … in a single transaction.
RecomputeSpeakerOrder(ctx context.Context, agendaPointID int64) error
```

### New meeting methods

```go
SetMeetingGenderQuotation(ctx context.Context, id int64, enabled bool) error
SetMeetingFirstSpeakerQuotation(ctx context.Context, id int64, enabled bool) error
// Pass nil to clear the moderator.
SetMeetingModerator(ctx context.Context, id int64, moderatorID *int64) error
```

### New agenda point methods

```go
// Pass nil to clear the override (revert to inheriting from meeting).
SetAgendaPointGenderQuotation(ctx context.Context, id int64, enabled *bool) error
SetAgendaPointFirstSpeakerQuotation(ctx context.Context, id int64, enabled *bool) error
// Pass nil to clear the moderator.
SetAgendaPointModerator(ctx context.Context, id int64, moderatorID *int64) error
```

> `SetSpeakerOrderPosition` (the raw per-row SQL update) is **not** on the interface — it is an internal detail used only by `RecomputeSpeakerOrder` inside the SQLite implementation.

---

## 3. Repository Implementation (`internal/repository/sqlite/repository.go`)

### Update `speakerFromClient` (or equivalent) helper

Map the four new SQLC fields onto the model:

```go
func speakerFromClient(row ...) *model.SpeakerEntry {
    return &model.SpeakerEntry{
        // ... existing fields ...
        GenderQuoted:  row.GenderQuoted,
        FirstSpeaker:  row.FirstSpeaker,
        Priority:      row.Priority,
        OrderPosition: row.OrderPosition,
    }
}
```

The same applies to the list variant used in `ListSpeakersForAgendaPoint`.

### Update meeting `fromClient` helper

```go
func meetingFromClient(row ...) *model.Meeting {
    return &model.Meeting{
        // ... existing fields ...
        GenderQuotationEnabled:       row.GenderQuotationEnabled,
        FirstSpeakerQuotationEnabled: row.FirstSpeakerQuotationEnabled,
        ModeratorID:                  nullInt64ToPtr(row.ModeratorID),
    }
}
```

### Update agenda point `fromClient` helper

`gender_quotation_enabled` and `first_speaker_quotation_enabled` are nullable BOOLEANs in SQLite, so SQLC generates `sql.NullBool`. Map to `*bool`:

```go
func agendaPointFromClient(row ...) *model.AgendaPoint {
    return &model.AgendaPoint{
        // ... existing fields ...
        GenderQuotationEnabled:       nullBoolToPtr(row.GenderQuotationEnabled),
        FirstSpeakerQuotationEnabled: nullBoolToPtr(row.FirstSpeakerQuotationEnabled),
        ModeratorID:                  nullInt64ToPtr(row.ModeratorID),
    }
}

// Helper (add near other null-conversion helpers):
func nullBoolToPtr(n sql.NullBool) *bool {
    if !n.Valid {
        return nil
    }
    return &n.Bool
}
```

### Update `AddSpeaker`

```go
func (r *SQLiteRepository) AddSpeaker(ctx context.Context, agendaPointID, attendeeID int64, speakerType string, genderQuoted, firstSpeaker bool) (*model.SpeakerEntry, error) {
    row, err := r.q.AddSpeaker(ctx, client.AddSpeakerParams{
        AgendaPointID: agendaPointID,
        AttendeeID:    attendeeID,
        Type:          speakerType,
        GenderQuoted:  genderQuoted,
        FirstSpeaker:  firstSpeaker,
    })
    if err != nil {
        return nil, err
    }
    return speakerFromClientRow(row), nil
}
```

### Implement `HasAttendeeSpokenOnAgendaPoint`

```go
func (r *SQLiteRepository) HasAttendeeSpokenOnAgendaPoint(ctx context.Context, agendaPointID, attendeeID int64) (bool, error) {
    return r.q.HasAttendeeSpokenOnAgendaPoint(ctx, client.HasAttendeeSpokenOnAgendaPointParams{
        AgendaPointID: agendaPointID,
        AttendeeID:    attendeeID,
    })
}
```

### Implement `SetSpeakerPriority`

```go
func (r *SQLiteRepository) SetSpeakerPriority(ctx context.Context, id int64, priority bool) error {
    return r.q.SetSpeakerPriority(ctx, client.SetSpeakerPriorityParams{
        ID:       id,
        Priority: priority,
    })
}
```

### Implement `RecomputeSpeakerOrder`

```go
func (r *SQLiteRepository) RecomputeSpeakerOrder(ctx context.Context, agendaPointID int64) error {
    rows, err := r.q.GetWaitingSpeakersForAgendaPoint(ctx, agendaPointID)
    if err != nil {
        return fmt.Errorf("RecomputeSpeakerOrder fetch: %w", err)
    }

    // Sort: priority DESC, gender_quoted DESC, first_speaker DESC, requested_at ASC
    sort.Slice(rows, func(i, j int) bool {
        a, b := rows[i], rows[j]
        if a.Priority != b.Priority {
            return a.Priority // true sorts before false
        }
        if a.GenderQuoted != b.GenderQuoted {
            return a.GenderQuoted
        }
        if a.FirstSpeaker != b.FirstSpeaker {
            return a.FirstSpeaker
        }
        return a.RequestedAt < b.RequestedAt // ISO8601 strings sort lexicographically
    })

    for i, row := range rows {
        if err := r.q.SetSpeakerOrderPosition(ctx, client.SetSpeakerOrderPositionParams{
            ID:            row.ID,
            OrderPosition: int64(i + 1),
        }); err != nil {
            return fmt.Errorf("RecomputeSpeakerOrder set position: %w", err)
        }
    }
    return nil
}
```

> If the repository has a `db *sql.DB` available (separate from `q`), wrap the loop in a transaction for atomicity.

### Implement meeting setters

```go
func (r *SQLiteRepository) SetMeetingGenderQuotation(ctx context.Context, id int64, enabled bool) error {
    return r.q.SetMeetingGenderQuotation(ctx, client.SetMeetingGenderQuotationParams{ID: id, GenderQuotationEnabled: enabled})
}

func (r *SQLiteRepository) SetMeetingFirstSpeakerQuotation(ctx context.Context, id int64, enabled bool) error {
    return r.q.SetMeetingFirstSpeakerQuotation(ctx, client.SetMeetingFirstSpeakerQuotationParams{ID: id, FirstSpeakerQuotationEnabled: enabled})
}

func (r *SQLiteRepository) SetMeetingModerator(ctx context.Context, id int64, moderatorID *int64) error {
    return r.q.SetMeetingModerator(ctx, client.SetMeetingModeratorParams{
        ID:          id,
        ModeratorID: ptrToNullInt64(moderatorID),
    })
}
```

### Implement agenda point setters

```go
func (r *SQLiteRepository) SetAgendaPointGenderQuotation(ctx context.Context, id int64, enabled *bool) error {
    return r.q.SetAgendaPointGenderQuotation(ctx, client.SetAgendaPointGenderQuotationParams{
        ID:                     id,
        GenderQuotationEnabled: ptrToNullBool(enabled),
    })
}

func (r *SQLiteRepository) SetAgendaPointFirstSpeakerQuotation(ctx context.Context, id int64, enabled *bool) error {
    return r.q.SetAgendaPointFirstSpeakerQuotation(ctx, client.SetAgendaPointFirstSpeakerQuotationParams{
        ID:                           id,
        FirstSpeakerQuotationEnabled: ptrToNullBool(enabled),
    })
}

func (r *SQLiteRepository) SetAgendaPointModerator(ctx context.Context, id int64, moderatorID *int64) error {
    return r.q.SetAgendaPointModerator(ctx, client.SetAgendaPointModeratorParams{
        ID:          id,
        ModeratorID: ptrToNullInt64(moderatorID),
    })
}
```

Add helpers near existing null-conversion helpers:

```go
func ptrToNullBool(p *bool) sql.NullBool {
    if p == nil {
        return sql.NullBool{}
    }
    return sql.NullBool{Bool: *p, Valid: true}
}
```

---

## 4. Fix Changed Call Site in `internal/handlers/manage.go`

`ManageSpeakerAdd` calls `h.Repository.AddSpeaker`. Update it to:

1. Load the meeting to get `GenderQuotationEnabled` and `FirstSpeakerQuotationEnabled`.
2. Load the agenda point to get its override values (`GenderQuotationEnabled`, `FirstSpeakerQuotationEnabled`).
3. Resolve effective setting: use agenda point value if `!= nil`, else use meeting value.
4. Load the attendee to get `Quoted`.
5. Compute `genderQuoted = attendee.Quoted && effectiveGenderQuotation`.
6. Call `HasAttendeeSpokenOnAgendaPoint` to compute `firstSpeaker`.
7. Call `AddSpeaker` with all parameters.
8. Call `RecomputeSpeakerOrder(ctx, agendaPointID)` after.

Also add `RecomputeSpeakerOrder` calls after the other speakers-list mutations that change the WAITING set:
- `ManageSpeakerRemove` — after `DeleteSpeaker`
- `ManageSpeakerStart` — after `SetSpeakerSpeaking` (speaker leaves WAITING)
- `ManageSpeakerEnd` — after `SetSpeakerDone` (no change to WAITING set, but positions may shift; call anyway for consistency)
- `ManageSpeakerWithdraw` — after `SetSpeakerWithdrawn`

---

## 5. Verification

```bash
task generate:db       # must exit 0 (already done in Phase 1, re-run if queries changed)
task build             # must exit 0 — all compile errors resolved
task test              # unit tests pass
task test:e2e          # all 52 E2E tests pass (new columns have safe defaults)
```
