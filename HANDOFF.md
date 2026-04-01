# Handoff

## Current Goal

Expand UI parity coverage in small, atomic steps that can each be:

- implemented independently
- fully verified
- regression-checked against the whole E2E suite
- documented in this file
- committed as a single-purpose change

The detailed expansion strategy lives in `ui-parity-expansion-plan.md`. The next execution mode is a commit-per-atomic-task workflow.

## Current State

`A01` through `A20` are complete locally, including `A11` agenda-point edit parity.

The parity expansion track is effectively complete. The active follow-up work is the
"Remove Legacy HTML Proxying" migration documented later in this file.

Current checkpoint (2026-04-01): Phase 5 is now implemented and fully verified.
Native join-QR and guest recovery pages are running through Connect RPCs, the
focused coverage passes, parity/legacy-contract coverage passes, and the full
E2E suite is green again after hardening the concurrent voting stress test.

### A01 — post-action parity helper

- added `compareFragmentAfterAction(...)` in `e2e/ui_parity_test.go`
- adopted the helper in `TestModerateSettingsTab_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`

### A04 — moderate parity: open vote panel

- added `TestModerateVotesPanelOpen_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- creates a draft vote via UI in each browser, opens it, then compares `#moderate-votes-panel` after removing transient `[data-notification-item]` elements via JS before comparison

### A05 — moderate parity: closed vote results

- added `TestModerateVotesPanelClosed_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- creates and opens a vote, closes it, waits for "Final Tallies" in both panels, removes notifications, then compares `#moderate-votes-panel`

### A06 — moderate attendee parity: add guest

- added `TestModerateAddGuestAttendee_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- adds a guest via the inline form in each browser, waits for card to appear, compares all `[data-testid='manage-attendee-card']` outer HTML

### A07 — moderate attendee parity: remove attendee

- added `TestModerateRemoveAttendee_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds Alice + Bob, removes Bob via remove button in each browser, waits for detach, compares remaining attendee cards

### A08 — attachment parity: populated attachment list

- added `TestAttachmentListPopulated_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- added `locatorAllInnerText` helper in `e2e/ui_parity_test.go`
- SPA uses `/blobs/:id/download`, legacy uses `/committee/:slug/meeting/:id/blob/:id` — different URL formats; compares `<a>` inner text content instead
- fixed SPA button text "Set Current" → "Set as Current" in `web/src/routes/committee/[committee]/meeting/[meetingId]/agenda-point/[agendaPointId]/tools/+page.svelte` to match i18n string

### A09 — current-document parity: selected attachment state

- added `TestCurrentDocumentState_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds an active meeting with agenda point, attachment, sets current via repo `SetCurrentAttachment`; member self-signs up and navigates to live page; verifies `[data-testid='live-doc-open-desktop']` and `[data-testid='live-doc-download-desktop']` present in both

### A10 — agenda parity: create agenda point

- added `TestModerateCreateAgendaPoint_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds one existing agenda point, creates `Budget Approval` via the moderation UI in each browser, waits for the new card to appear, then compares all `[data-testid='manage-agenda-point-card']` outer HTML
- aligned `e2e/current_doc_test.go` button selectors from `Set Current` to `Set as Current` so the existing current-document E2E test matches the already-updated UI label
- stabilized committee-page parity tests against nondeterministic meeting row ordering:
  - `TestCommitteeMeetingRows_UIParityWithLegacy` now sorts the captured row HTML before comparison
  - `TestCommitteeChairPage_UIParityWithLegacy` now compares sorted meeting-row HTML instead of the full `#meeting-list-container`

### A12 — agenda parity: reorder agenda point

- added `TestModerateReorderAgendaPoint_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds `First` and `Second`, moves `Second` up via the moderation UI in each browser, waits for it to become the first agenda card, then compares the ordered `[data-testid='manage-agenda-point-card']` outer HTML list

### A13 — agenda parity: delete agenda point

- added `TestModerateDeleteAgendaPoint_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds `Keep Me` and `Delete Me`, deletes `Delete Me` via the moderation UI in each browser, waits for the deleted card to detach, then compares the remaining `[data-testid='manage-agenda-point-card']` outer HTML list

### A20 — parity file organization review

- reviewed all three parity test files for structural issues
- `e2e/ui_parity_test.go` (684 lines): core helpers + basic page-level tests — well-structured, no action needed
- `e2e/ui_parity_extended_test.go` (1256 lines): feature-level tests with good section comments — navigable as-is; splitting would create import/build-tag boilerplate without real gain
- `e2e/ui_parity_legacy_contract_test.go` (333 lines): explicit legacy-contract documentation — purpose-built and appropriately scoped
- no structural changes made; all test names and comments accurately reflect their content
- no E2E run needed (no code changes)

### A19 — legacy fallback contract: vote partials and join-qr routes

- extended `e2e/ui_parity_legacy_contract_test.go` with three vote/manage legacy contract tests
- added `TestLegacyContract_VoteModeratorPartial`: both servers return identical `#moderate-votes-panel` for `GET /committee/.../votes/partial`
- added `TestLegacyContract_VoteLivePartial`: both servers return identical `#live-votes-panel` for `GET /committee/.../votes/live/partial`
- added `TestLegacyContract_JoinQRPage`: both servers serve the join-qr full page with a non-empty `#join-qr-code` img src
- updated file header to document these three route families as legacy-backed

Verification (2026-03-31): all 3 new tests PASS; full E2E suite PASS (all tests).

### A18 — legacy fallback contract: attendee-login/recovery routes

- extended `e2e/ui_parity_legacy_contract_test.go` with attendee-login contract tests
- added `TestLegacyContract_AttendeeLoginForm`: both servers return the same `<form>` for `GET /committee/.../attendee-login`
- added `TestLegacyContract_AttendeeLoginByLink`: both servers redirect to the live meeting page when `GET /committee/.../attendee-login?secret=<valid>`
- updated file header to document these routes as legacy-backed

### A17 — legacy fallback contract: docs routes

- created `e2e/ui_parity_legacy_contract_test.go` with package-level comment documenting the legacy-backed route contract pattern
- added `TestLegacyContract_DocsOOBFragment`: both servers return the same `hx-swap-oob` fragment for `/docs/oob/index`
- added `TestLegacyContract_DocsSearchPartial`: both servers return the same `#docs-search-results` container for `/docs/search?q=receipt`
- file includes instructions for removing entries when routes are ported away from the legacy handler

### A16 — speaker parity: end speaker

- added `TestModerateEndSpeaker_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds active agenda point + Alice + Bob; adds both, starts Alice, then ends Alice in each browser; waits for Bob in WAITING state; compares `#speakers-list-container` with `normalizeInitialScrollTop`
- fixed SPA done-speaker display number: added `doneDisplayNumber` function to `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` and used it in the `DONE` branch instead of `&nbsp;`
- fixed SPA waiting-speaker display number to include `doneCount + speakingCount` offset (matching legacy `WaitingDisplayNumber = doneCount + speakingCount + orderPosition`)
- added `normalizeInitialScrollTop` helper in `e2e/ui_parity_extended_test.go`; applied to A14, A15, A16 comparisons — `data-initial-scroll-top` is timing-dependent (static in SPA template vs. async JS in legacy)
- updated `TestSpeakersList_DoneSpeakerCanBeReadded` in `e2e/agenda_speakers_test.go` to expect the position number in the done-speaker column (now "1") instead of blank
- rebuilt `internal/web/build/`

### A15 — speaker parity: start speaker

- added `TestModerateStartSpeaker_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds an active agenda point and one attendee, adds `Alice Member` via the add-speaker dialog, then clicks `Start next speaker` in each browser, waits for `data-speaker-state='speaking'`, and compares `#speakers-list-container` with `normalizeSpeakingSinceAttr` applied

### A14 — speaker parity: add speaker

- added `TestModerateAddSpeaker_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- seeds an active agenda point and one attendee, adds `Alice Member` via the add-speaker dialog in each browser, waits for the speaker row to appear, then compares `#speakers-list-container`
- fixed SPA speaker parity mismatches:
  - `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` now computes waiting-speaker display numbers by counting WAITING speakers instead of using the unset `orderPosition` field
  - `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` and `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte` now use `First Time` to match legacy tooltip text
  - rebuilt `internal/web/build/`
- stabilized `TestModerateSpeakersWithAttendee_UIParityWithLegacy` by recomputing speaker order after direct repo seeding so both servers render the same waiting position

Verification completed (2026-03-30):

- all A04-A09 focused tests PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — all 28 PASS
- full E2E suite PASS (bz95ry5yu background run, prior to A08 SPA fix; parity suite re-verified after fix)

### A02 — live parity: active speaker state

- added `TestLiveActiveSpeaker_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- added `normalizeSpeakingSinceAttr` helper (strips `data-speaking-since` value and timer text) for cross-format normalization (Go Unix seconds vs JS `Date.now()` ms)
- fixed stale SPA build: rebuilt `internal/web/build/` to include `data-manage-scroll-anchor` attribute added in a prior SPA commit

### A03 — live parity: completed speaker state

- added `TestLiveCompletedSpeaker_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`
- fixed SPA `waitingDisplayNumber` bug in `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`: `SpeakerSummary` proto lacks `order_position`, so the old `speaker.orderPosition ?? 0` always returned 0; replaced with explicit 1-indexed counting over WAITING speakers (matches the moderate page approach)
- rebuilt SPA after the fix; `internal/web/build/` updated

Verification completed (2026-03-30):

- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestLiveActiveSpeaker_UIParityWithLegacy|TestLiveCompletedSpeaker_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — all 22 PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — all PASS

Verification completed (2026-03-31):

- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestCommitteeMeetingRows_UIParityWithLegacy|TestCommitteeChairPage_UIParityWithLegacy|TestModerateCreateAgendaPoint_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-03-31):

- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestModerateReorderAgendaPoint_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-03-31):

- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestModerateDeleteAgendaPoint_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-03-31):

- `nix develop -c bash -lc 'cd web && npm run build'` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestModerateAddSpeaker_UIParityWithLegacy|TestModerateSpeakersWithAttendee_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-03-31):

- `nix develop -c go test -v -tags=e2e -timeout=300s ./e2e/... -run "TestModerateStartSpeaker_UIParityWithLegacy"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — all 29 PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-04-01):

- `nix develop -c go test -v -tags=e2e -timeout=300s ./e2e/... -run "TestModerateEndSpeaker_UIParityWithLegacy|TestSpeakersList_DoneSpeakerCanBeReadded"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"` — all 30 PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-04-01):

- `nix develop -c go test -v -tags=e2e -timeout=300s ./e2e/... -run "TestLegacyContract_Docs"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy|TestLegacyContract"` — all 32 PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

Verification completed (2026-04-01):

- `nix develop -c go test -v -tags=e2e -timeout=300s ./e2e/... -run "TestLegacyContract_Attendee"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy|TestLegacyContract"` — all 34 PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

## Atomic Task Queue

Use the queue in `ui-parity-expansion-plan.md` under `Atomic Task Queue`.

Recommended execution order:

1. `A00` — baseline checkpoint commit
2. `A01` — post-action parity helper
3. `A02` — live parity active speaker state
4. `A03` — live parity completed speaker state
5. `A04` — moderate parity open vote panel
6. `A05` — moderate parity counted vote results
7. `A06` — attendee add-guest fragment parity
8. `A07` — attendee remove/update fragment parity
9. `A08` — attachment list parity
10. `A09` — current-document parity
11. `A10` — agenda create parity
12. `A11` — agenda edit parity
13. `A12` — agenda reorder parity
14. `A13` — agenda delete parity
15. `A14` — speaker add parity
16. `A15` — speaker start parity
17. `A16` — speaker end parity
18. `A17` — legacy fallback docs contract
19. `A18` — legacy fallback attendee-login/recovery contract
20. `A19` — legacy fallback vote/manage contract
21. `A20` — parity file organization cleanup if needed

## Atomic Task Protocol

Each atomic task should follow this sequence:

1. Implement the smallest possible change for that task only.
2. Run the smallest relevant focused test first.
3. If any `web/src/` file changed, run:
   `nix develop -c bash -lc 'cd web && npm run build'`
4. Run the full parity suite:
   `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"`
5. Run the full E2E suite:
   `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...`
6. Update this file with:
   - what changed
   - what passed
   - what the next task is
7. Create a commit for that task only.

## Recommended Next Task

Continue the native-SPA migration work:
- Phase 5: implement native Join QR and attendee recovery pages via Connect RPCs
- Phase 6: remove now-dead legacy HTML proxy branches from the E2E test server
- As follow-up cleanup, additional legacy-contract tests for POST routes can be removed or rewritten once those routes are fully ported

## Files Most Likely To Matter Next

- `ui-parity-expansion-plan.md`
- `e2e/ui_parity_test.go`
- `e2e/ui_parity_extended_test.go`
- `e2e/helpers_test.go`
- `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`
- `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`

## Notes For The Next Pass

- Prefer fragment parity over full-page parity.
- Reuse the existing dual-server browser harness instead of inventing a new test style.
- Keep normalization conservative.
- If a task requires SPA markup changes, rebuild before any E2E run.
- Do not mix multiple atomic tasks into one commit unless blocked by an unavoidable dependency.

---

# Migration: Remove Legacy HTML Proxying

## Directive (2026-04-01)

All HTML must be served natively from the SPA. Proxying HTML fragments from the legacy
HTMX/Templ handler into the SPA server is strictly forbidden. The legacy app remains
only for UI parity comparison tests and will eventually be removed entirely.

## Status

### Phase 1 — Documentation
- [x] Add no-HTML-proxy rule to CLAUDE.md (done — see "SPA Architecture Rule" section)

### Phase 2 — Attendee Connect RPCs ✅ DONE
- [x] Added `CreateAttendee`, `DeleteAttendee`, `SetChairperson`, `SetQuoted` RPCs to proto
- [x] Implemented in Go (`internal/services/attendees/service.go`, `internal/api/connect/attendee_handler.go`)
- [x] buf generate ran; Go + TS bindings generated
- [x] SPA updated: `addGuestAttendee`, `selfSignupAttendee`, `removeAttendee`, `toggleAttendeeChair`, `toggleAttendeeQuoted` all use Connect API
- [x] Removed `postLegacyAttendeeAction`, `attendeeCreateURL`, `attendeeSelfSignupURL`, `attendeeDeleteURL`, `attendeeToggleChairURL`, `attendeeToggleQuotedURL`, `legacyClientIDVals`, dead HTMX attributes

### Phase 3 — Agenda Point Edit (native Svelte) ✅ DONE
- [x] Added `UpdateAgendaPoint` RPC to `proto/conference/agenda/v1/agenda.proto`
- [x] Implemented in Go (`internal/services/agenda/service.go`, `internal/api/connect/agenda_handler.go`)
- [x] buf generate ran; Go + TS bindings generated
- [x] Replaced `hx-get` edit button with native inline Svelte edit form using `editingAgendaPointId`/`editingAgendaPointTitle` state
- [x] `startEditAgendaPoint`, `cancelEditAgendaPoint`, `saveEditAgendaPoint` functions added
- [x] `TestModerateEditAgendaPoint_UIParityWithLegacy` added and now passing

### Phase 4 — Vote Panel Native UI ✅ DONE (2026-04-01)
`loadLegacyVotesPanel()` fetched `/votes/partial` HTML. Replaced with fully native Svelte panel.

- [x] Removed `loadLegacyVotesPanelHTML` state variable (was never declared; confirmed already removed)
- [x] Removed `normalizeLegacyVoteOptionPlaceholders()` function
- [x] Fixed `selectModerateLeftTab()` — removed dead `legacyVotesPanelHTML` reference and `loadLegacyVotesPanel()` call
- [x] Removed `tick` from svelte import (was unused)
- [x] Replaced `{#if legacyVotesPanelHTML}{@html legacyVotesPanelHTML}{:else}...{/if}` placeholder with full native Svelte vote panel that renders from `votesState.data`:
  - Active vote card with live stats (eligible/cast/counted ballots) and Close button
  - Last-closed vote tally card with Archive button
  - Draft vote cards with editable options textarea, min/max inputs, Save draft and Open buttons
  - Create vote form: name input, open/secret radio, min/max, options textarea, Create button
  - Loading and error states wired to `votesState.loading` / `votesState.error`
- [x] All vote actions (`createVote`, `openVote`, `closeVote`, `archiveVote`, `saveDraftVote`) already used Connect API; now wired to the native panel buttons
- [x] Manual vote-entry flows stabilized after the native panel landed:
  - vote route shims now submit URL-encoded bodies with `HX-Request: true`
  - manual vote buttons call the submit helper explicitly instead of relying on implicit form submission
  - successful manual submissions reset the form before reloading panel state
  - single-select secret ballots now render radios instead of persistent checkboxes
  - archive actions surface a success notice (`Vote archived.`) in the SPA
- [x] Full-suite test stability improved by increasing `openModerateLeftTab(...)` waits in `e2e/browser_helpers_test.go`

Verification completed (2026-04-01):

- `nix develop -c bash -lc 'cd web && npm run build'` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestVoting_SecretVoteLifecycle_CountingAndVerificationGuards"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy|TestLegacyContract"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS

### Phase 5 — Join QR + Attendee Pages ✅ COMPLETE
Need Connect RPCs since these pages require backend secrets (meeting secret for join-qr, attendee secret for recovery).

- [x] Add `GetMeetingJoinQR` RPC to meetings proto
      Returns: `join_url`, `qr_code_data_url`, `meeting_name`, `committee_name`
      Auth: `moderate_access`
- [x] Add `GetAttendeeRecovery` RPC to attendees proto
      Returns: `login_url`, `qr_code_data_url`, `attendee_name`
      Auth: `moderate_access`
- [x] Implement both in Go services (reusing backend QR generation)
- [x] Run `buf generate` for Go + TS clients
- [x] Create SPA page: `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/join-qr/+page.svelte`
      Calls `GetMeetingJoinQR`, displays join URL link + QR image
- [x] Create SPA page: `web/src/routes/committee/[committee]/meeting/[meetingId]/attendee/[attendeeId]/recovery/+page.svelte`
      Calls `GetAttendeeRecovery`, displays login URL link + QR image
- [x] Verified `web/src/routes/committee/[committee]/meeting/[meetingId]/attendee-login/+page.svelte` is already native
      — It handles GET `?secret=` auto-login via `attendeeClient.attendeeLogin(...)`
      — It handles manual secret entry form via `attendeeClient.attendeeLogin(...)`
      — No legacy HTML dependency remains for the SPA implementation itself
- [x] Remove `shouldServeLegacyManageUtilityRoute` GET entries (join-qr, recovery)
- [x] Remove `shouldServeLegacyAttendeeLoginSecretRoute` entirely
- [x] Re-run focused verification after a clean SPA build:
      `TestManageJoinQRPage_ContainsSecretJoinURL|TestManagePage_GuestRecoveryLink|TestLegacyContract_JoinQRPage|TestLegacyContract_AttendeeLoginByLink`
      PASS on 2026-04-01 after clean rebuild
      Note: a prior overlapping run timed out waiting for `#join-qr-code` and should be ignored

### Phase 6 — Final Proxy Removal ⬜ TODO
- [x] Remove all `shouldServeLegacy*` functions from `e2e/helpers_test.go`
- [x] Remove their invocations from the `newTestServer()` switch statement
- [x] `legacyH`/`legacyRouter` comments now reflect the reduced proxy surface accurately
- [x] Run full E2E suite to verify nothing broke
- [x] Retire stale vote-partial legacy-contract tests from `e2e/ui_parity_legacy_contract_test.go`
      Vote partial endpoints are no longer treated as public legacy-contract routes in tests
      but the vote HTML endpoints still remain operationally legacy-backed in `newTestServer()`

Latest verification checkpoint (2026-04-01):

- `nix develop -c bash -lc 'cd web && npm run build'` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestManageJoinQRPage_ContainsSecretJoinURL|TestManagePage_GuestRecoveryLink|TestLegacyContract_JoinQRPage|TestLegacyContract_AttendeeLoginByLink"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy|TestLegacyContract"` — PASS
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS
- Regression fix applied:
      `e2e/voting_concurrent_test.go` now authenticates and submits concurrent ballots through the Connect RPCs (`AttendeeLogin`, `SubmitBallot`) used by the native app, while still preserving one cookie jar per voter.
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run TestVoting_Concurrent20Attendees_TallyIsCorrect` — PASS after the Connect-based test update
- Phase 6 checkpoint:
      `e2e/helpers_test.go` no longer defines or dispatches `shouldServeLegacyVoteRoute`, `shouldServeLegacyManageUtilityRoute`, or `shouldServeLegacyAgendaImportRoute`.
      The new E2E server now proxies `/docs/oob/...`, `/docs/search`, and the operational vote HTML endpoints through direct switch cases instead of the removed helper functions.
- Phase 6 correction:
      `e2e/ui_parity_legacy_contract_test.go` previously still asserted legacy vote-partial HTML for `/votes/partial` and `/votes/live/partial`.
      Those assertions are now removed so the contract file only documents route families we still want to treat as explicit legacy-backed contract surfaces.
- Live-vote migration status (2026-04-01, paused mid-step; not committed yet):
      The attendee live-vote panel is now rendered natively in `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte` from Connect data instead of calling `loadLegacyLiveVotesPanel()`.
      `proto/conference/votes/v1/votes.proto` now includes `LiveVoteCardView` plus `LiveVotePanelView.has_active_agenda` / `votes`; the generated Go/TS bindings were regenerated in-place.
      `internal/services/votes/service.go` now returns open votes, counting votes, and recently-closed timed-result cards through `GetLiveVotePanel`.
      `internal/api/connect/votes_handler_test.go` was expanded to cover the richer live panel states, including timed closed results and counting.
      The visible page no longer injects legacy vote HTML, but a hidden `liveVotesTemplateHTML()` placeholder and legacy-style HTMX attributes were intentionally restored only for parity/OOB markup compatibility inside the speakers container.
      Important local constraint: the worktree still contains unrelated in-flight edits in other files; this step stayed isolated to vote-panel files plus `HANDOFF.md`.
- Current verification for the paused live-vote step:
      `nix develop -c bash -lc 'PATH="$PATH:$PWD/web/node_modules/.bin" buf generate --template buf.gen.yaml --path proto/conference/votes/v1/votes.proto .'` — PASS
      `nix develop -c go test ./internal/api/connect -run 'TestVoteService_GetLiveVotePanel_AndSubmitBallot|TestVoteService_GetLiveVotePanel_IncludesClosedTimedResultsAndCountingState'` — PASS
      `nix develop -c bash -lc 'cd web && npm run build'` — PASS after the latest live-page changes; `internal/web/build` written
      `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run 'TestVoting_OpenVote_ModeratorAndAttendeeHappyPath_HTMX|TestVoting_SecretVoteLifecycle_CountingAndVerificationGuards|TestVoting_LivePanelUpdatesViaSSEOnVoteOpen|TestVoting_DuplicateSecretSubmissionRejected'` — PASS after rebuilding
      `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run 'TestMeetingLive_UIParityWithLegacy'` — PASS after rebuilding
- Still pending before this step can be committed:
      Re-run `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run '.*UIParityWithLegacy|TestLegacyContract'` from the latest build
      Re-run `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...`
      If both pass, update this section again and commit the isolated live-vote migration step
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy|TestLegacyContract"` — PASS after restoring direct vote-route switch cases
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...` — PASS after restoring direct vote-route switch cases
- Next step after the paused verification: finish the broad-suite re-check for the native live-vote panel, then move on to porting the remaining vote form-post endpoints off the legacy HTML routes

## Reference: Proxy Route Status (updated 2026-04-01)

### `shouldServeLegacyAgendaImportRoute`
| Route | SPA status |
|-------|-----------|
| GET `/moderate/agenda` | Dead code — SPA renders inline |
| GET `/agenda-point/list` | Dead code — SPA uses Connect |
| GET `/agenda-point/{id}/edit-form` | **PORTED** — native Svelte edit form (Phase 3 ✅) |
| POST `/agenda/import/extract` | Dead code — `event.preventDefault()` + native JS |
| POST `/agenda/import/diff` | Dead code — `event.preventDefault()` + native JS |
| POST `/agenda-point/{id}/move-up` | Dead code — Connect API |
| POST `/agenda-point/{id}/move-down` | Dead code — Connect API |
| POST `/agenda-point/{id}/edit` | **PORTED** — `agendaClient.updateAgendaPoint()` (Phase 3 ✅) |
→ All cases dead. Can remove `shouldServeLegacyAgendaImportRoute` in Phase 6.

### `shouldServeLegacyVoteRoute`
| Route | SPA status |
|-------|-----------|
| GET `/votes/partial` | Compatibility endpoint only — not covered as a legacy contract anymore, but still used by legacy vote-form responses |
| GET `/votes/live/partial` | **IN TRANSITION** — visible attendee panel now renders natively from `GetLiveVotePanel`, but a hidden compatibility template + legacy-style HTMX attrs still reference this URL until broad-suite re-verification is complete |
| POST `/votes/*` | **ACTIVE** — current E2E browser flows still exercise legacy HTMX vote form endpoints |
→ The old `shouldServeLegacyVoteRoute` helper is gone, but the vote HTML routes themselves are not fully removable yet.
→ Current migration focus: finish verifying the native attendee panel migration, then port the remaining vote POST / ballot flows so the vote HTML routes can be removed cleanly.

### `shouldServeLegacyManageUtilityRoute`
| Route | SPA status |
|-------|-----------|
| GET `/moderate/join-qr` | **ACTIVE** — link navigates there (Phase 5) |
| GET `/attendee/{id}/recovery` | **ACTIVE** — link navigates there (Phase 5) |
| POST `/attendee/create` | **PORTED** — `attendeeClient.createAttendee()` (Phase 2 ✅) |
| POST `/attendee/self-signup` | **PORTED** — `attendeeClient.selfSignup()` (Phase 2 ✅) |
| POST `/attendee/{id}/delete` | **PORTED** — `attendeeClient.deleteAttendee()` (Phase 2 ✅) |
| POST `/attendee/{id}/chair` | **PORTED** — `attendeeClient.setChairperson()` (Phase 2 ✅) |
| POST `/attendee/{id}/quoted` | **PORTED** — `attendeeClient.setQuoted()` (Phase 2 ✅) |
→ Only GET join-qr and GET recovery remain. Both need new Connect RPCs (Phase 5).

### `shouldServeLegacyAttendeeLoginSecretRoute`
| Route | SPA status |
|-------|-----------|
| GET/POST `/attendee-login` (with secret) | **ACTIVE** — attendee login flow (Phase 5) |
→ `AttendeeLogin` RPC already exists; SPA page at `web/src/.../attendee-login/` needs to handle `?secret=` auto-login and manual form.
→ Helper removed in Phase 5; this section is retained only as historical migration context.
