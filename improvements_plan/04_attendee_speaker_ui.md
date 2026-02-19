# Phase 4 — Attendee-Facing Speaker Request UI

## Goal

Update the attendee-facing meeting view so attendees can see their position in the speakers list (including quotation status and first-speaker indicator), and receive real-time order updates via SSE when the list changes. This phase may be deferred until the chairperson workflow (Phase 3) is fully stable.

---

## Prerequisites

- Phase 1–3 complete and verified.
- The project's SSE broker (`internal/broker/`) is already in place.
- The attendee session middleware (`internal/middleware/attendee_session.go`) is already in place.

---

## 1. Speakers List SSE Event

### When to publish

Publish an SSE event named `speakers-updated` whenever the speakers list for an agenda point changes. The event payload should be the agenda point ID (so clients can ignore events for agenda points they are not watching).

Publish from the repository layer or from the handlers that mutate the speakers list. The cleanest approach: add a call to `h.Broker.Publish(...)` at the end of each handler that mutates speakers (`ManageSpeakerAdd`, `ManageSpeakerRemove`, `ManageSpeakerStart`, `ManageSpeakerEnd`, `ManageSpeakerWithdraw`, `ManageSpeakerTogglePriority`, `ManageMeetingSetQuotation`, `ManageAgendaPointSetQuotation`).

### SSE payload

```
event: speakers-updated
data: {"agenda_point_id": 42}
```

---

## 2. Attendee Meeting Page Template

The attendee-facing meeting page (likely `internal/templates/meeting_join.templ` or similar) should display:

- The current WAITING speakers in order (ordered by `order_position`).
- For each entry: name, type, gender-quoted indicator, first-speaker badge.
- The attendee's own position in the queue (highlighted row).

Use HTMX SSE extension to refresh the list when a `speakers-updated` event arrives:

```html
<div hx-ext="sse"
     sse-connect="{ SSEEndpointURL }"
     sse-swap="speakers-updated"
     hx-target="#attendee-speakers-list"
     hx-select="#attendee-speakers-list">
  <div id="attendee-speakers-list">
    <!-- speakers list partial rendered here -->
  </div>
</div>
```

The SSE endpoint returns a full re-render of the speakers list partial when the event fires.

---

## 3. Attendee `quoted` Field Visibility

Currently `attendee.quoted` is set from `user.quoted` at join time. Consider:

- Should an attendee be able to see their own `quoted` status on the join/meeting page?
- Should a chairperson be able to toggle an attendee's `quoted` status from the management page?

If the answer to either is yes, add:
- A read-only display of `quoted` on the attendee's own view.
- A chairperson toggle: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/quoted` → `ManageAttendeeSetQuoted`.

Note: changing `attendee.quoted` does **not** retroactively change the `gender_quoted` field on existing speakers-list entries (it is a snapshot). It only affects future `AddSpeaker` calls.

---

## 4. Routes to Add

```yaml
# SSE stream for live speakers list updates (attendee-facing)
- path: /committee/{slug}/meeting/{meeting_id}/speakers/stream
  sse: true
  methods:
    - verb: GET
      handler: AttendeeSpeakersStream
      middleware: [session, attendee_session]

# (optional) Toggle attendee quoted status
- path: /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/quoted
  methods:
    - verb: POST
      handler: ManageAttendeeSetQuoted
      middleware: [session, auth, committee_access]
      template:
        package: github.com/Y4shin/conference-tool/internal/templates
        type: AttendeeListPartial
        input_type: AttendeeListPartialInput
```

---

## 5. Handler Sketches

### `AttendeeSpeakersStream`

```go
func (h *Handler) AttendeeSpeakersStream(w http.ResponseWriter, r *http.Request, params routes.RouteParams) error {
    ch := h.Broker.Subscribe(r.Context())
    flusher, ok := w.(http.Flusher)
    if !ok {
        return fmt.Errorf("streaming not supported")
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    for {
        select {
        case <-r.Context().Done():
            return nil
        case event := <-ch:
            if event.Name == "speakers-updated" {
                fmt.Fprintf(w, "event: speakers-updated\ndata: %s\n\n", event.Data)
                flusher.Flush()
            }
        }
    }
}
```

---

## 6. Verification

```bash
task generate:routes
task generate:templates
task build
task test
task test:e2e
```

### New E2E tests to add in `e2e/attendee_speakers_test.go`

| Test name | Behaviour |
|-----------|-----------|
| `TestAttendee_SpeakersListUpdates_ViaSSE` | Attendee joins meeting; chairperson adds a speaker; verify attendee page updates without reload |
| `TestAttendee_SeesOwnPositionHighlighted` | Attendee requests to speak; verify their row is visually distinguished |
| `TestAttendee_QuotedBadgeVisible` | Attendee with `quoted = true` sees their quoted indicator in their speakers list row |
