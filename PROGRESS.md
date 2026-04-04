# Bug Fixes & Improvements Progress

## Status: In Progress

---

## Bugs

### 1. Login page card extends to bottom — DONE
- **File**: `web/src/routes/login/+page.svelte`
- **Fix**: Replaced `h-full` with `min-h-full` and `align-center` (invalid) with `items-center` for proper vertical centering. Moved error alert inside the fieldset for consistency.

### 2. Admin login page styling & back button — DONE
- **Files**: `web/src/routes/admin/login/+page.svelte`, `web/messages/en.json`, `web/messages/de.json`
- **Fix**: Wrapped admin login form in matching fieldset/legend card styling. Added "Back to Login" button. Added `admin_login_legend` and `admin_login_back_button` i18n keys.

### 3. Moderate attendee row layout — DONE
- **File**: `web/src/lib/components/ui/AttendeeRow.svelte`
- **Fix**: Moved Chair/FLINTA* toggles from a separate row below buttons into the same inline flex row (hidden on small screens). Changed toggles to `toggle-xs` for compact layout.

### 4. QR code SVG icon malformed — DONE
- **File**: `web/src/lib/components/ui/LegacyIcon.svelte`
- **Fix**: Replaced the broken QR code SVG path with a cleaner Material Symbols variant that renders the three finder patterns and data modules correctly.

### 5. Past speakers don't show speaking time — DONE
- **File**: `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/SpeakersSection.svelte`
- **Fix**: Replaced `doneDisplayNumber()` (position index) with `formatDuration(speaker.durationSeconds)` to show mm:ss speaking time for DONE speakers.

### 6. Help page dark mode images not switching — DONE
- **File**: `web/src/routes/+layout.svelte`
- **Fix**: `applyTheme()` now writes a `conference-tool-theme` cookie (`light`/`dark`) alongside localStorage. The backend already reads this cookie in `VariantFromRequest()` to resolve image variants, but the frontend was never setting it.

### 7. Speaker search adds wrong participant — DONE
- **File**: `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/SpeakersSection.svelte`
- **Fix**: `handleSpeakerSearchEnter()` now reads the input value directly from `event.target` and recomputes the candidate ranking inline, avoiding stale `bind:value` state that could cause the wrong attendee to be selected.

### 8. Speaker ordering — FLINTA* always on top instead of interleaved — DONE
- **File**: `internal/repository/sqlite/repository.go`
- **Fix**: Rewrote `RecomputeSpeakerOrder()` to look up the effective gender quotation setting and, when enabled, interleave FLINTA* and non-FLINTA* speakers in round-robin fashion (FLINTA* first). Within each gender group, speakers are sorted by priority, first-speaker status, then request time. Updated E2E test expectations in `speakers_quotation_test.go`.

### 9. Voting form allows wrong number of choices — DONE
- **Files**: `internal/services/votes/service.go`, `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`
- **Fix**: Added min/max selection count validation to `SubmitBallot()`, `CountOpenBallot()`, and `CountSecretBallot()` on the backend. Frontend submit button is now disabled when selection count is out of range, and selection hints are shown using existing i18n keys.

### 10. GOPATH issue in nix flake — DONE
- **File**: `flake.nix`
- **Fix**: Added `export GOPATH` to shellHook to override the broken relative `GOPATH=go` from `~/.config/go/env` (managed by home-manager).

---

## Features (from IMPROVEMENTS.md)

*(not yet started)*

- New Meeting Wizard
- QR Codes as Dialogs
- Receipts show actual voting behaviour
- Receipts in the meeting view page
- Record when agenda points are entered and left
- Admin pages layout cleanup
