# Agent Handoff

## Purpose

This file captures the current frontend rewrite state so another agent can continue Phase 5 without re-discovering the recent work.

Date: 2026-03-26
Repo state at handoff creation: clean after commit `d4cd535`

## Current Rewrite Status

Source of truth:

- [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md)
- [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md)

Plan snapshot:

- Current phase: `Phase 5 - Full Feature Port`
- Phase 0: completed
- Phase 1: completed
- Phase 2: completed
- Phase 3: completed
- Phase 4: completed
- Phase 5: in progress
- Phase 6: not started

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

Still not at parity:

- full voting parity
  - draft editing
  - secret ballot submission/counting
  - richer moderator tally/verification flows
  - remaining vote verification surfaces
- admin/account management parity
- docs/public verification parity
- attachments and file-serving flows
- later Phase 6 legacy removal

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

