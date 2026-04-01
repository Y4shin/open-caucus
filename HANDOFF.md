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

`A01` through `A10`, plus `A12`–`A20`, are complete locally.

`A11` is currently blocked on missing agenda-point edit functionality in the product/UI surface.

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

All planned parity tasks (A01–A20, excluding blocked A11) are complete.

The next natural area of expansion would be:
- **A21**: agenda-point edit parity — currently blocked on missing edit UI in the product; revisit when edit functionality is implemented
- Additional legacy-contract tests for POST routes (vote create, attendee actions) if those routes are ported away from the legacy handler
- Parity coverage for the attendee-facing live page under various vote states

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
