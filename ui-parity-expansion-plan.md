# UI Parity Expansion Plan

## Goal

Expand legacy-vs-SPA parity testing so it protects the migration at the level that matters most:

- high-value route states
- post-interaction fragment updates
- SSE-driven view changes
- explicitly documented legacy fallback boundaries

The plan is to deepen parity coverage without creating a separate testing style. New work should continue to use the existing dual-server browser harness that compares the SPA-backed app against the legacy HTMX/Templ server.

## Status Snapshot

Last updated: 2026-03-30

- Current dedicated parity coverage: `21` browser parity tests
- Current parity files:
  - `e2e/ui_parity_test.go`
  - `e2e/ui_parity_extended_test.go`
- Current parity mechanism:
  - boot a new app server with SPA + Connect
  - boot a legacy server with HTMX + Templ
  - seed equivalent data into both
  - drive equivalent browser actions
  - compare normalized HTML fragments
- Current strength:
  - good coverage for major screens and a subset of key fragments
- Current gap:
  - limited state-matrix coverage
  - limited parity after mutations
  - limited explicit coverage for legacy fallback routes still served inside the new app

## Why This Matters

The migration risk is no longer just “does the page load.” The larger risks now are:

- a state variant renders differently than the legacy UI
- a mutation updates the wrong fragment shape after an HTMX-like interaction
- live and moderation screens drift when SSE/refetch behavior changes
- a route still depends on the legacy handler, but that dependency is implicit rather than tested

The parity suite should act as a migration contract, not just a small smoke test.

## Principles

### Keep The Existing Harness

Use `newTestServer()` and `newLegacyTestServer()` from `e2e/helpers_test.go` as the standard path for all parity additions.

### Prefer Fragment Parity Over Full-Page Parity

Compare the smallest meaningful DOM slice:

- cards
- rows
- forms
- panels
- swapped containers

This reduces noise and makes failures easier to diagnose.

### Compare Real States, Not Just Initial Render

A parity test should often:

1. seed a meaningful state
2. perform the same user interaction in both apps
3. wait for the target fragment to settle
4. compare the resulting fragment HTML

### Avoid Over-Normalizing

The current normalization helpers are useful, but they should not erase meaningful differences in:

- element presence
- form structure
- attributes that affect behavior
- badge/state labels
- ordering

If new normalization is added, it should be justified by a known framework artifact rather than general mismatch tolerance.

### Use Parity To Guide Legacy Removal

When a fallback route is still served by the legacy handler in the “new” app, parity coverage should make that explicit so removal work has a checklist.

## Current Coverage Baseline

The current parity suite already covers:

- login page
- admin login page
- home page
- committee overview for chair workflows
- active-meeting card for member workflows
- admin dashboard, accounts, and committee-user pages
- meeting join and attendee login screens
- live meeting page
- moderate workspace shell
- docs overlay, docs search, and receipts
- committee meeting rows
- moderation subpanels for agenda, attendees, votes, and settings
- speaker-list parity in selected live and moderate states

This is a good base, but it is still selective. The next phase should expand coverage by state and by interaction.

## Scope Of Expansion

### Workstream 1: State-Matrix Coverage

Add parity tests for important route variants that already exist functionally but are not yet covered strictly against legacy HTML.

Priority states:

- committee overview
  - member vs chairperson
  - no meetings vs multiple meetings
  - active meeting vs no active meeting
  - signup open vs signup closed
- join and attendee login
  - anonymous guest open state
  - anonymous guest closed state
  - logged-in member join state
  - attendee-login validation/error state where practical
- live meeting
  - no active agenda point
  - active agenda point with no speakers
  - queued speakers
  - active speaker
  - completed speaker history if legacy UI shows it
  - vote panel hidden/empty/open/closed/result states
- moderate workspace
  - empty attendees
  - populated attendees
  - empty speaker queue
  - queued speakers
  - active speaker
  - active agenda point vs none
  - votes panel empty/configured/open/countable/counted
  - meeting settings vs agenda settings sub-tabs
- admin/docs/public
  - empty table states where applicable
  - populated states where applicable
  - docs search with result vs no result
  - receipts empty state vs populated state if supported

### Workstream 2: Post-Interaction Fragment Parity

Add parity tests that compare the DOM after user-driven mutations rather than only initial GET responses.

Priority interactions:

- committee and join flows
  - self-signup transition from join page to live page
- moderate attendees
  - add guest attendee
  - attendee row updates after change/removal where supported in SPA
- agenda management
  - create agenda point
  - edit agenda point
  - reorder agenda point
  - delete agenda point
- speakers
  - add speaker
  - start next speaker
  - end current speaker
  - quoted/priority/moderator badge states
- votes
  - create vote draft
  - open vote
  - submit attendee ballot
  - close vote
  - count vote
- docs
  - help overlay open state
  - docs search result container
- attachments
  - attachment list after upload
  - current-document panel after selecting an attachment

The preferred pattern is:

- perform the action in both browsers
- wait for the exact fragment to update
- compare only that fragment

## High-Value Test Additions

The first six additions should be:

1. Live page parity with an active speaker and a completed speaker.
2. Moderate page parity while a vote is open and after vote results exist.
3. Attendee-management fragment parity after add/remove actions.
4. Attachment and current-document parity once files exist.
5. Agenda editor parity after create/edit/reorder/delete mutations.
6. Legacy fallback contract coverage for routes still intentionally served by the legacy handler.

## Atomic Execution Model

Future implementation work should be broken into atomic tasks with these constraints:

- one user-visible parity outcome per task
- one main domain area per task where possible
- green focused tests before broad verification
- green full parity suite before commit
- green full E2E suite before commit
- rebuild the SPA before verification if any `web/src/` file changed
- update `HANDOFF.md` at the end of each task before committing

### Standard Verification Cadence

Use this sequence for each atomic task:

1. Run the smallest relevant focused parity test or focused E2E slice.
2. If SPA files changed, run `nix develop -c bash -lc 'cd web && npm run build'`.
3. Run the full parity suite:
   `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"`
4. Run the full E2E suite:
   `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...`
5. Update `HANDOFF.md` with:
   - what was completed
   - what remains
   - how the next task should start
6. Create a commit for that single task.

### Baseline Note

The current workspace already contains verified parity-fix changes from the previous pass. Before starting new expansion work, create a baseline commit from the current green state so the remaining tasks can each be committed independently.

## Atomic Task Queue

The tasks below are intentionally small enough to implement, verify, regression-check, document in `HANDOFF.md`, and commit one by one.

| ID | Atomic task | Outcome | Likely files | Focused verification |
| --- | --- | --- | --- | --- |
| `A00` | Baseline checkpoint commit | Capture the already-green parity-fix work and planning docs as the starting point for future atomic commits | `e2e/ui_parity_extended_test.go`, `web/src/routes/committee/[committee]/+page.svelte`, `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`, `web/src/routes/committee/[committee]/meeting/[meetingId]/join/+page.svelte`, `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`, `ui-parity-expansion-plan.md`, `HANDOFF.md` | Full parity suite and full E2E suite |
| `A01` | Add parity helper for post-action fragment comparison | Reduce duplication for action-then-compare tests | `e2e/ui_parity_test.go`, possibly `e2e/helpers_test.go` | Run helper-adopting parity tests or one focused parity test using the helper |
| `A02` | Live parity: active speaker state | Compare live speaker list when one speaker is actively speaking | `e2e/ui_parity_extended_test.go` or `e2e/ui_parity_live_test.go` | Focused live parity test |
| `A03` | Live parity: completed speaker state | Compare live speaker list/history after a speaker has finished | same as `A02` | Focused live parity test |
| `A04` | Moderate parity: open vote panel | Compare moderate votes panel while a vote is open | `e2e/ui_parity_extended_test.go` or `e2e/ui_parity_moderate_test.go` | Focused moderate vote parity test |
| `A05` | Moderate parity: counted vote results | Compare moderate votes panel after closing/counting a vote | same as `A04` | Focused moderate vote-results parity test |
| `A06` | Moderate attendee parity: add guest | Compare attendee panel fragment after adding a guest attendee | `e2e/ui_parity_extended_test.go`, possibly helper file | Focused attendee-management parity test |
| `A07` | Moderate attendee parity: remove or update attendee row | Compare changed attendee fragment after a removal or supported update action | same as `A06` | Focused attendee row parity test |
| `A08` | Attachment parity: populated attachment list | Compare attachment/tooling UI once uploaded attachments exist | new parity file or existing extended file | Focused attachment parity test |
| `A09` | Current-document parity: selected attachment state | Compare current-document panel after setting the current attachment | same as `A08` | Focused current-document parity test |
| `A10` | Agenda parity: create agenda point | Compare agenda list fragment after creating a point | `e2e/ui_parity_extended_test.go` or `e2e/ui_parity_moderate_test.go` | Focused agenda-create parity test |
| `A11` | Agenda parity: edit agenda point | Compare edited agenda-point card or list fragment | same as `A10` | Focused agenda-edit parity test |
| `A12` | Agenda parity: reorder agenda points | Compare reordered agenda list after move action | same as `A10` | Focused agenda-reorder parity test |
| `A13` | Agenda parity: delete agenda point | Compare agenda list after deletion | same as `A10` | Focused agenda-delete parity test |
| `A14` | Speaker parity: add speaker from moderation UI | Compare speaker queue fragment after adding a speaker | `e2e/ui_parity_extended_test.go` or `e2e/ui_parity_moderate_test.go` | Focused add-speaker parity test |
| `A15` | Speaker parity: start next speaker | Compare speaker queue after transitioning a queued speaker into active state | same as `A14` | Focused start-speaker parity test |
| `A16` | Speaker parity: end current speaker | Compare speaker queue/history after ending the current speaker | same as `A14` | Focused end-speaker parity test |
| `A17` | Legacy fallback contract: docs routes | Explicitly lock down `/docs/oob/...` and `/docs/search` legacy-backed behavior | new `e2e/ui_parity_legacy_contract_test.go` or similar | Focused legacy-contract docs test |
| `A18` | Legacy fallback contract: attendee login/recovery routes | Explicitly lock down attendee-login/recovery legacy-backed behavior | same as `A17`, possibly `e2e/helpers_test.go` | Focused legacy-contract attendee test |
| `A19` | Legacy fallback contract: vote/manage utility routes | Explicitly lock down vote-form and manage-utility legacy-backed behavior | same as `A17` | Focused legacy-contract vote/manage test |
| `A20` | Parity file organization cleanup | Split parity tests by domain if the suite has become hard to maintain | parity test files only | Full parity suite and full E2E suite |

### Optional Follow-On Atomic Tasks

These are useful if the first queue lands cleanly:

- anonymous guest join-page closed-state parity
- attendee-login validation-error parity
- docs no-results parity
- admin empty-state parity
- receipts populated-state parity

## Coverage Matrix

| Area | Current parity status | Recommended additions | Likely files |
| --- | --- | --- | --- |
| Committee overview | partial | chair/member state matrix, empty state, active meeting variants | `e2e/ui_parity_test.go`, `e2e/ui_parity_extended_test.go` |
| Join and attendee login | partial | closed/open guest states, richer validation and form variants | `e2e/ui_parity_test.go`, `e2e/ui_parity_extended_test.go` |
| Live meeting | partial | active speaker, completed speaker, vote lifecycle states | `e2e/ui_parity_extended_test.go` or split live-specific parity file |
| Moderate workspace | partial | agenda mutations, attendee mutations, active speaker controls, vote lifecycle states | `e2e/ui_parity_extended_test.go` or split moderate-specific parity file |
| Admin | partial | empty-state parity where meaningful | `e2e/ui_parity_test.go` |
| Docs and receipts | partial | no-results search, populated receipts variants | `e2e/ui_parity_test.go` |
| Attachments/current document | missing | uploaded attachment list and current-doc panel parity | new parity test file or `e2e/ui_parity_extended_test.go` |
| Legacy fallback routes | implicit | explicit contract test coverage | `e2e/helpers_test.go`, new parity/contract test file |

## Legacy Fallback Contract

The new app still intentionally serves some routes through the legacy handler path inside `newTestServer()`. That is acceptable during migration, but it should be tested explicitly.

Current categories called out in `e2e/helpers_test.go` include:

- vote-form related legacy routes
- attendee manage utility routes
- agenda import routes
- attendee-login/recovery routes
- docs fragment routes such as `/docs/oob/...`
- docs search HTML route

### Plan

- Add a dedicated contract-style test file that documents which route families are still expected to be legacy-backed.
- For each route family, verify:
  - it is intentionally routed through the legacy path today
  - the response structure still matches the expected legacy fragment contract
- When a route family is ported, remove it from this contract list and add direct parity or behavioral SPA coverage instead.

This prevents “hidden legacy dependency” drift.

## Harness Improvements

The current parity helpers are already strong. A small amount of helper work would make expansion cheaper and more consistent.

Recommended helper additions:

- `compareFragmentAfterAction(...)`
  - run an action in both pages, wait for a selector, compare HTML
- `waitForStableHTML(...)`
  - poll until HTML stops changing for dynamic containers that briefly churn
- `assertEqualLocatorSets(...)`
  - compare ordered lists of repeated row/card fragments
- `gotoAndWaitForVisibleSelector(...)`
  - distinguish attached vs visible states where tabs or dialogs matter
- targeted helpers for moderate workflows:
  - open right-side tools tab
  - open votes panel state
  - open agenda settings sub-tab

Normalization improvements should stay conservative. Good candidates are:

- transitional HTMX classes
- Svelte comment nodes
- attribute ordering

Bad candidates are:

- removing meaningful data attributes
- rewriting text content
- collapsing optional elements that represent real structure

## File Organization Plan

If parity coverage keeps growing, split by domain rather than keeping everything in one extended file.

Recommended shape:

- `e2e/ui_parity_test.go`
  - shell-level and broad route coverage
- `e2e/ui_parity_committee_test.go`
  - home, committee, join, attendee login
- `e2e/ui_parity_live_test.go`
  - live page states, votes, speakers
- `e2e/ui_parity_moderate_test.go`
  - moderation tabs, agenda, attendees, speakers, votes
- `e2e/ui_parity_admin_docs_test.go`
  - admin, docs, receipts
- `e2e/ui_parity_legacy_contract_test.go`
  - routes still intentionally backed by legacy handlers

This split is optional at first. If only a few tests are added, continuing in `e2e/ui_parity_extended_test.go` is fine.

## Implementation Phases

### Phase 1: Highest-ROI State Additions

Add parity for:

- live page with active speaker
- live page with completed speaker
- moderate votes panel with an open vote
- moderate votes panel with counted results
- attendee management panel after adding a guest
- attachment/current-document populated states

Acceptance criteria:

- at least one new parity test per target state
- no fragile dependence on full-page wrappers
- tests pass under `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParity"`

### Phase 2: Mutation-Driven Fragment Parity

Add interaction parity for:

- agenda create/edit/reorder/delete
- attendee add/remove
- speaker add/start/end
- vote create/open/submit/close/count

Acceptance criteria:

- each mutation compares the fragment that changed
- both new and legacy flows are driven through equivalent browser actions where practical
- failures identify the fragment under comparison clearly

### Phase 3: Legacy Fallback Contract

Add explicit tests for route families still served by the legacy path.

Acceptance criteria:

- the list of intentional legacy-backed routes is documented in tests
- new fallback categories cannot appear silently
- route families are removed from the contract as they are ported

### Phase 4: Cleanup And Maintenance

- extract shared parity helpers if duplication grows
- split parity tests by domain if file size becomes unwieldy
- remove tests tied to retired legacy-only surfaces only after equivalent SPA coverage exists

## Recommended First Implementation Order

1. Add live-page active/completed speaker parity.
2. Add moderate votes open/results parity.
3. Add attendee add/remove fragment parity.
4. Add attachment/current-document parity.
5. Add agenda mutation parity.
6. Add legacy fallback contract tests.

This order balances migration risk against implementation cost and should surface the highest-value regressions first.

## Verification

Use the Nix shell for parity verification so the Go and browser toolchain are available consistently.

Suggested commands:

```bash
nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"
nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...
```

If SPA files change while implementing parity fixes or parity-enabling markup:

```bash
nix develop -c bash -lc 'cd web && npm run build'
```

## Done Criteria

This plan is complete when all of the following are true:

- parity coverage includes the main route-state matrix for committee, join, live, and moderate flows
- at least the most important mutation-driven fragments are compared after interaction
- live and moderation vote/speaker states have stronger parity protection
- legacy fallback boundaries are explicit and tested
- the suite remains readable, diagnosable, and fast enough to use during migration work
