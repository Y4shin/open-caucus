# Bug Fixes & Improvements Progress

## Status: In Progress (New Meeting Wizard remaining)

---

## Bugs — All Fixed

1. **Login page vertical centering** — `align-center` → `items-center`, `min-h-full`
2. **Admin login styling & back button** — Card styling matches regular login, back button added
3. **Attendee row toggle layout** — Chair/FLINTA* toggles moved inline with action buttons
4. **QR code SVG icon** — Replaced malformed path with clean Material Symbols QR code
5. **Past speakers speaking time** — Shows mm:ss duration instead of position number
6. **Help page dark mode images** — Theme cookie set on preference change for backend variant resolution
7. **Speaker search off-by-one** — Reads input value directly from event target
8. **Speaker ordering interleaving** — FLINTA*/non-FLINTA* round-robin instead of all-FLINTA*-first
9. **Voting form choice validation** — min/max selection enforcement on backend + disabled submit button on frontend
10. **GOPATH in nix flake** — shellHook sets GOPATH to override broken relative path

## Features — Completed

### QR Codes as Dialogs
- Join QR and recovery QR now open as dialog modals on the moderate page instead of navigating to separate pages
- Both dialogs include clipboard copy button for the URL

### Record Agenda Point Timestamps
- New `entered_at` and `left_at` columns on agenda_points (migration 031)
- Service sets timestamps when activating/deactivating agenda points
- Sidebar shows entry time; edit dialog shows duration

### Receipts Show Voting Behaviour
- Secret ballot verification now returns `choice_labels` by querying `vote_ballot_selections`
- Frontend receipts page displays choices identically for open and secret ballots

### Receipts in Meeting View
- "My Receipts" button in votes section of meeting live view
- Dialog shows receipts matching the current meeting's votes with inline verification

### Admin Pages Layout Cleanup
- Responsive grid forms (sm:grid-cols), consistent label/input styling
- Action columns right-aligned with proper button groups
- Consistent card headings, centered pagination, removed inline pipe separators

## Features — Remaining

### New Meeting Wizard
- Multi-step creation wizard: basic data → agenda editor → participant import → overview/confirm
