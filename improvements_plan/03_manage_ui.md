# Phase 3 — Chairperson Management UI ✅ DONE

## Goal

Expose the new data-layer capabilities (quotation toggles, moderator, priority) in the chairperson meeting-management page. All interactions must use HTMX partial swaps — no full page reloads.

---

## 1. New Routes (`routes.yaml`)

Add the following routes. Run `task generate:routes` after editing.

```yaml
# Toggle priority for a WAITING speaker
- path: /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/priority
  methods:
    - verb: POST
      handler: ManageSpeakerTogglePriority
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: SpeakersListPartial
        input_type: SpeakersListPartialInput

# Meeting-level quotation toggle
- path: /committee/{slug}/meeting/{meeting_id}/quotation
  methods:
    - verb: POST
      handler: ManageMeetingSetQuotation
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: SpeakersListPartial
        input_type: SpeakersListPartialInput

# Meeting-level moderator
- path: /committee/{slug}/meeting/{meeting_id}/moderator
  methods:
    - verb: POST
      handler: ManageMeetingSetModerator
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: MeetingSettingsPartial
        input_type: MeetingSettingsPartialInput

# Agenda point quotation override
- path: /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/quotation
  methods:
    - verb: POST
      handler: ManageAgendaPointSetQuotation
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: SpeakersListPartial
        input_type: SpeakersListPartialInput

# Agenda point moderator
- path: /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/moderator
  methods:
    - verb: POST
      handler: ManageAgendaPointSetModerator
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: AgendaPointPartial
        input_type: AgendaPointPartialInput
```

> Adjust template types if the project uses different partial names. The key constraint is that each handler returns only the affected partial — not the full page.

After editing routes.yaml:

```bash
task generate:routes
```

---

## 2. Handler Changes

### New file: `internal/handlers/speakers_settings.go` (or add to `manage.go`)

#### `ManageSpeakerTogglePriority`

1. Load the speaker entry by `params.SpeakerId`.
2. Toggle `priority`: call `SetSpeakerPriority(ctx, id, !entry.Priority)`.
3. Call `RecomputeSpeakerOrder(ctx, entry.AgendaPointID)`.
4. Return the updated `SpeakersListPartialInput` (same as other speakers handlers).

#### `ManageMeetingSetQuotation`

1. Parse form fields: `gender_quotation_enabled` (bool), `first_speaker_quotation_enabled` (bool).
2. Call `SetMeetingGenderQuotation` and/or `SetMeetingFirstSpeakerQuotation` as appropriate.
3. If the current agenda point is set, call `RecomputeSpeakerOrder` for it (settings affect ordering).
4. Return the updated `SpeakersListPartialInput`.

#### `ManageMeetingSetModerator`

1. Parse form field: `moderator_id` (int64 or empty string for clearing).
2. Call `SetMeetingModerator(ctx, meetingID, moderatorIDPtr)`.
3. Return the updated meeting-settings partial.

#### `ManageAgendaPointSetQuotation`

1. Parse form fields: `gender_quotation_enabled` and `first_speaker_quotation_enabled` (each may be "true", "false", or "" to inherit).
2. Call `SetAgendaPointGenderQuotation` / `SetAgendaPointFirstSpeakerQuotation` with `*bool` or `nil`.
3. Call `RecomputeSpeakerOrder(ctx, agendaPointID)`.
4. Return the updated `SpeakersListPartialInput`.

#### `ManageAgendaPointSetModerator`

1. Parse form field: `moderator_id`.
2. Call `SetAgendaPointModerator`.
3. Return the updated agenda-point partial.

---

## 3. Template Changes (`internal/templates/meeting_manage.templ`)

Run `task generate:templates` after editing.

### `SpeakersListPartialInput` — add new fields

```go
type SpeakersListPartialInput struct {
    CommitteeSlug                string
    IDString                     string
    CurrentAgendaPointID         *int64
    Speakers                     []SpeakerItem
    Attendees                    []AttendeeItem
    GenderQuotationEnabled       bool   // meeting-level
    FirstSpeakerQuotationEnabled bool   // meeting-level
    // resolved effective settings for the current agenda point:
    EffectiveGenderQuotation       bool
    EffectiveFirstSpeakerQuotation bool
    Error                          string
}
```

### `SpeakerItem` — add new fields

```go
type SpeakerItem struct {
    ID           int64
    IDString     string
    AttendeeName string
    Type         string
    Status       string
    IsWaiting    bool
    IsSpeaking   bool
    GenderQuoted bool   // display indicator
    FirstSpeaker bool   // display badge
    Priority     bool   // show as highlighted / show toggle state
}
```

### Speakers list table — add columns and priority toggle

In the speakers table in `SpeakersListPartial`:

1. Add a "Q" (quoted) indicator column showing `gender_quoted`.
2. Add a "1st" badge column showing `first_speaker`.
3. For WAITING speakers, add a priority toggle button:
   ```html
   <form hx-post="{ SpeakerPriorityPostStr(s) }"
         hx-target="#speakers-list-container"
         hx-swap="outerHTML">
     <button type="submit">{ if s.Priority }★{ else }☆{ end }</button>
   </form>
   ```

### Quotation toggles — in the speakers section header or meeting settings

Add a small form above the speakers table (targeting the speakers container):

```html
<form hx-post="{ QuotationPostStr() }" hx-target="#speakers-list-container" hx-swap="outerHTML">
  <label>
    <input type="checkbox" name="gender_quotation_enabled"
           checked?={ input.EffectiveGenderQuotation }> Gender quotation
  </label>
  <label>
    <input type="checkbox" name="first_speaker_quotation_enabled"
           checked?={ input.EffectiveFirstSpeakerQuotation }> First-speaker bonus
  </label>
  <button type="submit">Apply</button>
</form>
```

### Moderator dropdown — in meeting settings and/or per-agenda-point section

Similar to the protocol writer dropdown. A `<select>` populated from `Attendees` with an empty option for "none".

---

## 4. Verification

```bash
task generate:routes
task generate:templates
task build            # must exit 0
task test             # unit tests pass
task test:e2e         # all existing tests pass
```

### New E2E tests to add in `e2e/speakers_quotation_test.go`

| Test name | Behaviour |
|-----------|-----------|
| `TestSpeakers_PriorityToggle_MovesToFront` | Add two speakers; toggle priority on the second; verify they appear first in the list |
| `TestSpeakers_GenderQuotation_OrdersQuotedFirst` | Disable gender quotation then re-enable; verify quoted speaker appears before non-quoted |
| `TestSpeakers_FirstSpeaker_Badge` | Seed one speaker who has spoken before and one who hasn't; verify first-speaker badge only on the new one |
| `TestSpeakers_MeetingModerator_SetAndClear` | Set and then clear the meeting moderator; verify UI updates without full page reload |
