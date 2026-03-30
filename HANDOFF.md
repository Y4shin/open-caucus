# Handoff

## Current Goal

Fix all 9 failing parity tests in `e2e/ui_parity_extended_test.go`, then verify the entire E2E suite is green.

## Background

The project is a Go + HTMX/Templ legacy server being migrated to a SvelteKit SPA. The legacy stack is kept alive as a comparison target for UI parity tests. The parity tests boot both servers, render the same pages, and assert that normalized HTML matches.

The SPA build is embedded in the Go binary at startup — **after any SPA file change you must run `cd web && npm run build` before re-running E2E tests**.

## What Was Completed Before This Checkpoint

- Removed raw SSE path; moved meeting streaming to Connect RPC.
- Moved docs page/search to Connect.
- Moved remaining `/api` JSON routes to Connect.
- Added a `render-template` command for rendering legacy Templ components from JSON.
- Fixed E2E suite after legacy fallback removal (added `newLegacyTestServer` comparison path).
- Added and committed `e2e/ui_parity_extended_test.go` with 9 new parity tests (all currently failing).

## Current State

All 9 tests in `e2e/ui_parity_extended_test.go` **fail**. No code has been changed toward fixing them yet. The root causes are well understood (see below).

## How To Run Tests

```bash
# Run only the new extended parity tests (fastest iteration loop):
go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestCommitteeMeetingRows_UIParityWithLegacy|TestCommitteeActiveMeetingCard_UIParityWithLegacy|TestModerateAgendaEditor_UIParityWithLegacy|TestModerateAttendeesTab_UIParityWithLegacy|TestModerateVotesPanel_UIParityWithLegacy|TestModerateSettingsTab_UIParityWithLegacy|TestModerateSpeakersWithAttendee_UIParityWithLegacy|TestMeetingLiveWithSpeakersInQueue_UIParityWithLegacy|TestMeetingJoinFullForm_UIParityWithLegacy|TestAttendeeLoginFullForm_UIParityWithLegacy"

# Run all parity tests:
go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"

# Run full E2E suite:
go test -v -tags=e2e -timeout=600s ./e2e/...

# Build SPA (needed after any SPA file change):
cd web && npm run build
```

## Failing Tests: Root Causes and Fixes

### Fix 1 — `TestCommitteeMeetingRows_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:15`

**Root cause**: The test is seeded with a chairperson. The meeting rows show a signup-open toggle; the SPA reads `item.meeting?.signupOpen` from the `MeetingReference` proto field, which IS populated by the backend. This test may actually pass now — **verify first** by running it. If it fails, look for HTML attribute ordering or extra/missing attributes in the rendered `<li>` elements by reading the actual diff output.

**Selector under test**: `[data-testid='committee-meeting-row']`

---

### Fix 2 — `TestCommitteeActiveMeetingCard_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:43`

**Root cause**: The test logs in as `chair1` (chairperson). In the legacy Templ template (`internal/templates/committee.templ`, line ~256), `committee-active-meeting-card` is rendered inside an `else` branch that only executes when `canManage == false`. Chairpersons have `canManage=true`, so they never see this card. Likewise in the SPA (`web/src/routes/committee/[committee]/+page.svelte`), the card is only rendered for non-chair users.

**Fix (test-only)**: Seed a `member` role user (e.g. `member1`/`pass123`) and log in as that user instead of `chair1`. The active meeting card will then be visible on both sides.

---

### Fix 3 — `TestModerateAgendaEditor_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:105`

**Root cause**: The test asserts `locatorAllOuterHTML(t, ..., "#agenda-point-list-container li")`. In the SPA (`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`, line ~1580), agenda point cards inside `#agenda-point-list-container` are `<div data-testid="manage-agenda-point-card">`, not `<li>`. The legacy also uses `<div data-testid="manage-agenda-point-card">` (see `internal/templates/meeting_manage.templ`).

**Fix (test-only)**: Change selector from `"#agenda-point-list-container li"` to `"[data-testid='manage-agenda-point-card']"` on both sides.

---

### Fix 4 — `TestModerateVotesPanel_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:151`

**Root cause**: The test navigates to the moderate page and immediately waits for `#moderate-votes-panel`. In the SPA, the votes panel is nested inside the "Tools" tab which is not the default active tab. The element is in the DOM but hidden; the legacy may render it differently.

**Fix (test-only)**: After navigating to the moderate page, open the tools tab before asserting:
1. Activate the agenda point (it's seeded as "Main Topic") — needed so the votes panel is relevant.
2. Open the tools tab by clicking the button that shows tools. Look for a tab button with `data-moderate-right-tab='tools'` or similar in `moderate/+page.svelte` around line 1800.
3. Wait for `#moderate-votes-panel` to be visible, then assert.

Alternatively, use `locator.waitFor({state: 'attached'})` vs `{state: 'visible'}` to check the hidden panel is structurally identical even when not visible.

---

### Fix 5 — `TestModerateSettingsTab_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:182`

**Root cause**: The test opens the settings left-tab and asserts `#moderate-speaker-settings-container`. In the SPA (`moderate/+page.svelte`, line ~1989), this element is inside the "Agenda Point" settings sub-tab. The default active sub-tab appears to be "Meeting". So the element may not be present until the user clicks the "Agenda Point" sub-tab.

**Fix (test-only)**: After opening the settings left-tab, additionally click the "Agenda Point" sub-tab button. Look for `[data-moderate-settings-tab='agenda']` or similar in the moderate SPA.

---

### Fix 6 — `TestModerateSpeakersWithAttendee_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:219`

**Root cause**: Speaker quick-control buttons (start/end speaking) in the SPA `#speakers-list-container` are plain `<button onclick={...}>` elements. The legacy (`internal/templates/meeting_manage.templ`) wraps each quick-control action in a `<form hx-post="...">` element. The HTML structure therefore differs between legacy and SPA.

**Fix options** (pick one):
- **SPA fix** (preferred for long-term parity): Wrap the speaker quick-control buttons in `<form>` elements with `use:legacyAttrs` providing `hx-post` to `/committee/{slug}/meeting/{meetingId}/speaker/{speakerId}/start` and `/end`, mirroring the legacy. This ensures the HTML is structurally identical.
- **Test fix** (simpler short-term): Instead of comparing the full `#speakers-list-container`, compare only the speaker item metadata (`[data-speaker-state]`, `[data-testid='live-speaker-name']`) and exclude the quick-control buttons from comparison.

The legacy URL patterns: `SpeakerStartPostStr = "/committee/{slug}/meeting/{meetingId}/speaker/{speakerId}/start"` and `/end`.

**SPA file**: `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` around line 2021 (`#speakers-list-container` quick controls).

---

### Fix 7 — `TestMeetingLiveWithSpeakersInQueue_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:265`

**Root cause**: The legacy `LiveSpeakerCard` template (`internal/templates/meeting_live.templ`, line ~300) adds `data-manage-scroll-anchor="false"` to each `<li>`. The SPA live page (`web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`, lines 806-811) does **not** include this attribute.

**Fix (SPA)**: Add `data-manage-scroll-anchor="false"` (hardcoded `false` for attendee view — chairpersons use `true` for the active speaker when moderating) to each `<li data-testid="live-speaker-item">` in the SPA live page, around line 807.

After the SPA fix, rebuild: `cd web && npm run build`.

---

### Fix 8 — `TestMeetingJoinFullForm_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:336`

**Root cause**: The test asserts `locatorOuterHTML(t, ..., "main")`. The `<main>` element wraps the entire page layout including the app shell (header, nav) which has different Tailwind class ordering or extra attributes between SPA layout and legacy template wrapper.

**Fix (test-only)**: Replace the `"main"` selector with a more specific selector that targets only the page-content section. In the join SPA (`web/src/routes/committee/[committee]/meeting/[meetingId]/join/+page.svelte`), the content is wrapped in a `<div class="space-y-6">` or similar inner div. Use `"main > div"` or compare specific sub-elements like the form directly (`"main form"`).

---

### Fix 9 — `TestAttendeeLoginFullForm_UIParityWithLegacy`

**File**: `e2e/ui_parity_extended_test.go:366`

**Root cause**: Same as Fix 8 — `"main"` includes the layout wrapper which differs between SPA and legacy.

**Fix (test-only)**: Use `"main form"` or `"main > div"` as the selector instead of `"main"`.

---

## Key Files

| File | Purpose |
|------|---------|
| `e2e/ui_parity_extended_test.go` | All 9 failing parity tests |
| `e2e/helpers_test.go` | `testServer` setup, seed helpers, `openModerateAgendaEditor`, `openModerateLeftTab` |
| `web/src/routes/committee/[committee]/+page.svelte` | Committee SPA (meeting rows, active-meeting-card) |
| `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte` | Live meeting SPA (speaker items, agenda stack) |
| `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` | Moderate SPA (agenda editor, votes panel, settings, speakers) |
| `web/src/routes/committee/[committee]/meeting/[meetingId]/join/+page.svelte` | Join page SPA |
| `web/src/routes/committee/[committee]/meeting/[meetingId]/attendee-login/+page.svelte` | Attendee login SPA |
| `internal/templates/committee.templ` | Legacy committee template |
| `internal/templates/meeting_live.templ` | Legacy live meeting template |
| `internal/templates/meeting_manage.templ` | Legacy moderate/manage template |

## Recommended Fix Order

1. **Test-only fixes first** (no SPA rebuild needed): Fix 2, 3, 5, 8, 9
2. **Verify Fix 1** by running `TestCommitteeMeetingRows` — may already pass
3. **Fix 4** (test-only, needs investigation into moderate tab selectors)
4. **SPA fixes** (require `cd web && npm run build` after each): Fix 6 (speakers forms), Fix 7 (scroll anchor attr)

## Verification

```bash
# After all fixes, run full suite:
go test -v -tags=e2e -timeout=600s ./e2e/...
```

All tests should be green before this work is considered done.
