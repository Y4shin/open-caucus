# Agent Handoff

## Purpose

This file captures the current frontend rewrite state so another agent can continue Phase 6 legacy-removal work without re-discovering the recent Phase 5 completion work. It is being updated incrementally during active work so another agent can pick up mid-slice if quota runs out.

Date: 2026-03-27
Repo state at handoff update: dirty working tree with completed Phase 5 implementation plus active Phase 6 server split, SPA parity fixes, and E2E migration/debugging

## Current Rewrite Status

Source of truth:

- [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md)
- [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md)

Plan snapshot:

- Current phase: `Phase 6 - Legacy Removal`
- Phase 0: completed
- Phase 1: completed
- Phase 2: completed
- Phase 3: completed
- Phase 4: completed
- Phase 5: completed
- Phase 6: in progress

What is already ported into the SPA:

- session bootstrap/login/logout
- committee home and committee overview
- join flow and attendee login
- live meeting read model
- moderation read model
- signup-open moderation control
- speaker queue controls on moderation and live pages
- moderation attendee search and add-speaker flow
- first agenda-management slice
  - top-level create
  - activate/deactivate
  - reorder
  - delete
- first voting slice
  - moderator votes panel
  - draft creation
  - open vote
  - close vote
  - archive closed vote
  - attendee open-ballot submission

Phase 5 parity now includes:

- full voting parity for the currently supported open and secret-ballot workflows
- admin/account management parity
- docs/public verification parity
- attachments and file-serving flows

What remains:

- Phase 6 legacy removal
- replacing HTML-over-SSE/HTMX-only refresh paths with pure typed invalidation flows
- removing the old SSR route/template stack once the mapping and API coverage are sufficient

## Latest Phase 6 Direction

The current migration direction is:

- keep `serve` as SPA + typed API only
- move the legacy HTMX/Templ app to a separate `serve-legacy` subcommand
- migrate browser E2E coverage to the new SPA implementation now, even while `serve-legacy` still exists for manual comparison/debugging

This replaced the earlier mixed-hosting approach because the combined mux caused root-route conflicts and made parity debugging harder.

## What Landed Today

Server/runtime split:

- added a shared serve setup in [`cmd/serve_common.go`](/mnt/c/Users/Patric/Projects/conference-tool/cmd/serve_common.go)
- converted [`cmd/serve.go`](/mnt/c/Users/Patric/Projects/conference-tool/cmd/serve.go) toward SPA/API-only serving
- added [`cmd/serve_legacy.go`](/mnt/c/Users/Patric/Projects/conference-tool/cmd/serve_legacy.go) for the old app

SPA serving and auth parity:

- fixed built asset lookup in [`internal/web/handler.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/web/handler.go) by trimming the leading slash before reading from the embedded build FS
- added coverage in [`internal/web/handler_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/web/handler_test.go)
- added SPA admin login at [`web/src/routes/admin/login/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/admin/login/+page.svelte)
- added auth-provider/session bootstrap parity through:
  - [`proto/conference/session/v1/session.proto`](/mnt/c/Users/Patric/Projects/conference-tool/proto/conference/session/v1/session.proto)
  - [`internal/services/session/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/session/service.go)
  - [`web/src/lib/stores/session.svelte.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/stores/session.svelte.ts)

Backend/API parity fixes:

- added shared moderation/chair authorization helpers in [`internal/services/authz/authz.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/authz/authz.go)
- wired those checks into:
  - [`internal/services/moderation/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/moderation/service.go)
  - [`internal/services/attendees/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/attendees/service.go)
  - [`internal/services/votes/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/votes/service.go)
  - [`internal/services/agenda/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/agenda/service.go)
- restored the “members may only open the active meeting” rule in [`internal/services/meetings/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/meetings/service.go)
- aligned the API test setup with that rule in [`internal/api/connect/meeting_handler_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/meeting_handler_test.go) by explicitly marking the seeded meeting active

Moderation page parity work:

- [`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte) now includes:
  - parent selection for sub-agenda point creation
  - legacy-compatible ids/data-testids for agenda cards and actions
  - agenda import dialog parity hooks and client-side import/diff/accept/deny behavior
  - button-based clickable import correction rows to satisfy Svelte a11y checks
  - `data-testid="manage-speakers-card"` wrapper restored for speakers E2E scoping
  - no-active-agenda-point speaker empty state restored

E2E migration/hardening:

- [`e2e/helpers_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/helpers_test.go) and [`e2e/oauth_helpers_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/oauth_helpers_test.go) now boot the SPA/API-oriented server shape instead of relying on the legacy router
- [`e2e/browser_helpers_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/browser_helpers_test.go) was updated to:
  - tolerate SPA moderation layouts without the old left-tab controls
  - wait for SPA agenda content directly
  - harden login flows with `gotoAndWaitForInput(...)` retries for `/login` and `/admin/login`
- [`e2e/attendee_speakers_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/attendee_speakers_test.go) now uses the same hardened helper for attendee login
- SPA-semantic E2E updates landed in:
  - [`e2e/access_control_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/access_control_test.go)
  - [`e2e/attendee_login_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/attendee_login_test.go)
  - [`e2e/moderate_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/moderate_test.go)
  - [`e2e/oauth_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/oauth_test.go)
  - [`e2e/admin_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/e2e/admin_test.go)

## Current Verification Snapshot

Confirmed passing in this workstream:

- `nix develop . --command go test ./...`
- `nix develop . --command npm run check`
- `nix develop . --command npm run build`
- targeted E2E slices for:
  - admin access-control redirects
  - moderation access
  - attendee login/logout
  - agenda CRUD/import flows

Full E2E status at latest update:

- the suite progressed through access-control, admin, and agenda/import tests after the login-helper hardening
- the next concrete failure discovered was `TestSpeakersList_NoActivePoint`
- that failure was traced to missing SPA compatibility hooks in the moderation speakers card:
  - missing `data-testid="manage-speakers-card"` wrapper
  - wrong empty-state text when no agenda point is active
- both fixes have now been applied in [`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte)
- targeted rerun of the speakers slice was in progress at handoff time and should be rerun first

## Immediate Next Steps

If you are resuming from here:

1. Rerun:
   - `nix develop . --command npm run check` from `web/`
   - `nix develop . --command env PLAYWRIGHT_DRIVER_PATH=... PLAYWRIGHT_BROWSERS_PATH=... PLAYWRIGHT_NODEJS_PATH=... go test -v -tags=e2e -timeout=180s ./e2e/... -run 'TestSpeakersList_(NoActivePoint|AddSpeaker|SearchEnterAddsBestMatch|OneNonDoneEntryPerType|StartEnd)'`
2. If that speakers slice passes, rerun the full E2E suite against the SPA server.
3. Continue fixing the next parity gap surfaced by the suite before removing more legacy code.
4. Update this file again after each substantial fix or rerun so another agent can always resume cleanly.

## Most Recent Commits

Recent rewrite commits, newest first:

- `d4cd535` `feat(web): add typed voting workflow to moderate and live pages`
- `0972109` `feat(web): add agenda management to moderation workspace`
- `467d64f` `feat(web): add moderation attendee search for speakers`
- `3f87594` `feat(web): add speaker queue controls to live and moderate`
- `fe2a8a0` `feat(web): port attendee join and login flows`
- `415bff9` `feat(web): add first Phase 4 SPA meeting slice`
- `bef27cd` `feat(web): complete Phase 3 frontend foundation`

## Important Files

Primary SPA surfaces:

- [`web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte)
  - live meeting page
  - attendee self-add speaker controls
  - live vote panel and open-ballot submission
- [`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte)
  - moderation summary
  - speaker queue controls
  - attendee search/add speaker
  - agenda management
  - moderator votes panel

Typed client entrypoints:

- [`web/src/lib/api/index.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/api/index.ts)
- [`web/src/lib/api/services.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/api/services.ts)

Realtime invalidation:

- [`web/src/lib/utils/sse.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/utils/sse.ts)
  - listens for `attendees.updated`, `speakers.updated`, `agenda.updated`, `votes.updated`, plus older meeting refresh events

Backend vote/agenda service layer:

- [`internal/services/agenda/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/agenda/service.go)
- [`internal/services/votes/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/votes/service.go)

API integration coverage:

- [`internal/api/connect/agenda_handler_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/agenda_handler_test.go)
- [`internal/api/connect/votes_handler_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/votes_handler_test.go)
- [`internal/api/connect/testserver_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/testserver_test.go)

Contracts:

- [`proto/conference/agenda/v1/agenda.proto`](/mnt/c/Users/Patric/Projects/conference-tool/proto/conference/agenda/v1/agenda.proto)
- [`proto/conference/votes/v1/votes.proto`](/mnt/c/Users/Patric/Projects/conference-tool/proto/conference/votes/v1/votes.proto)

## What Landed In The Agenda Slice

Commit: `0972109`

Frontend:

- moderation page now loads `AgendaService.ListAgendaPoints`
- moderation page can:
  - create top-level agenda points
  - activate/deactivate the current point
  - move top-level points up/down
  - delete points

Backend/API tests:

- added move/reorder coverage
- added permission coverage for activation

Docs:

- mapping matrix updated with agenda create/activate/reorder/delete rows
- rewrite plan updated to mention the agenda slice and new immediate next steps

Known limitation:

- this is the first agenda slice, not full agenda parity
- sub-point editing/import/tools/attachments remain outside this slice

## What Landed In The Voting Slice

Commit: `d4cd535`

Frontend:

- moderation page now loads `VoteService.GetVotesPanel`
- moderation votes card supports:
  - create draft vote
  - open vote
  - close vote
  - archive closed vote
- live meeting page now loads `VoteService.GetLiveVotePanel`
- live vote card supports:
  - attendee-side open-ballot submission
  - already-voted feedback
  - ineligible feedback
  - receipt token feedback after submit

Backend/API tests:

- new file [`internal/api/connect/votes_handler_test.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/votes_handler_test.go)
- added coverage for:
  - create/open/close happy path
  - attendee live vote bootstrap and ballot submission
  - member forbidden from vote creation

Shared test helpers:

- `combinedTestServer` now has helpers for seeding and activating agenda points

Docs:

- mapping matrix updated with the first voting rows
- rewrite plan updated to mention the initial open-ballot voting slice

Known limitations:

- no SPA draft-editing flow yet
- secret ballot submission is not wired in the SPA
- moderator counting/verification flows are not ported
- live vote UX currently targets the open-ballot path only

## Suggested Next Steps

Best next target:

1. Finish voting parity before jumping to admin/docs.

Recommended order:

1. Draft editing on moderation page
2. Secret ballot flow
3. Moderator tally/counting/verification UI
4. Remaining vote-related API integration coverage
5. Then move to admin/account management
6. Then docs/public verification
7. Then attachments/file-serving flows

Why:

- voting is already partially ported and now has momentum
- the plan file already calls out remaining voting work explicitly
- the live and moderation pages both now depend on `VoteService`, so finishing that area reduces the most obvious parity gap

## Verification Commands Recently Used

Run commands inside the Nix devshell:

```bash
nix develop . --command sh -lc 'cd web && npm run check'
nix develop . --command sh -lc 'cd web && npm run build'
nix develop . --command go test ./internal/api/connect/...
nix develop . --command go test ./...
```

These all passed at the end of the last voting slice before this handoff file was created.

## Environment Notes

- `node`, `go`, and `buf` are available in the Nix devshell
- approved command prefix already exists for:
  - `nix develop . --command`
- manual edits should use `apply_patch`
- prefer `rg` for searching
- commit messages in this repo should be detailed:
  - short subject
  - body paragraph
  - flat bullet list

There is still a devshell bootstrap quirk where entering `nix develop` prints:

```text
/tmp/nix-shell.XXXXXX: line 2172: $'\r': command not found
```

In my sessions this did not block the actual command being run. `npm run check`, `npm run build`, and `go test` still completed successfully afterward.

## Current Git State

At handoff creation:

- worktree clean
- HEAD: `d4cd535`

## Pickup Advice For The Next Agent

If you are continuing immediately, start here:

1. Re-read [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md) and keep it as the source of truth.
2. Re-read [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md) before removing or replacing legacy workflows.
3. Continue with voting parity on the existing SPA pages before creating new route surfaces unless the plan requires a new route.
4. Keep verification tight after each slice:
   - `npm run check`
   - `npm run build`
   - `go test ./internal/api/connect/...`
   - `go test ./...`
