# Agent Handoff

## Purpose

This file captures the current frontend rewrite state so another agent can continue Phase 6 legacy-removal work. Updated after Phase 6b cleanup completion.

Date: 2026-03-28
Repo state: dirty working tree — full E2E suite passes (14.3s)

## Current Rewrite Status

Source of truth:

- [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md)
- [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md)

Plan snapshot:

- Current phase: `Phase 6 - Legacy Removal`
- Phase 0–5: completed
- Phase 6a (decoupling): completed — `serve.go` and `serve_common.go` fully decoupled from `internal/handlers`
- Phase 6b (Connect streaming): **completed and verified**

What is already ported into the SPA:

- session bootstrap/login/logout
- committee home with full chairperson management (create/delete/toggle-active meetings)
- committee member view (active meeting card + join button)
- join flow and attendee login
- live meeting read model
- moderation read model
- signup-open moderation control
- speaker queue controls on moderation and live pages
- moderation attendee search and add-speaker flow
- full agenda management (create/activate/deactivate/reorder/delete/import)
- full voting parity (moderator + attendee, open + secret ballot)
- admin/account management parity
- docs/public verification parity
- attachments and file-serving flows
- **Connect streaming RPC for meeting events** (replaces raw SSE + refetch pattern)

## Phase 6b: Connect Streaming — COMPLETED

**What was implemented** (all verified — `go build ./...`, `go test ./...`, E2E all pass):

### Proto
- `proto/conference/meetings/v1/meetings.proto`: added `SubscribeMeetingEvents` RPC, `SubscribeMeetingEventsRequest` message, `MeetingEventKind` enum (UNSPECIFIED/SPEAKERS_UPDATED/VOTES_UPDATED/AGENDA_UPDATED/ATTENDEES_UPDATED/MEETING_UPDATED), `MeetingEvent` message
- `buf generate` run — Go + TypeScript clients regenerated

### Backend
- `internal/api/connect/meeting_handler.go`: `MeetingHandler` now takes `broker.Broker`; `NewMeetingHandler(service, broker)` updated; `SubscribeMeetingEvents` method streams typed events by mapping broker event names to `MeetingEventKind`; sends initial `MEETING_UPDATED` to confirm connection
- Updated callers: `cmd/serve.go`, `e2e/helpers_test.go`, `e2e/oauth_helpers_test.go`, `internal/api/connect/testserver_test.go`

### Frontend
- `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`:
  - Removed `connectEventStream` import and `refreshTick` state
  - Added Connect streaming `$effect` using `meetingClient.subscribeMeetingEvents()`
  - Dispatches to `loadLiveMeeting()`, `loadSpeakers()`, `loadVotes()` based on event kind
  - Added three silent targeted loader functions (no loading spinner flash)
  - Post-mutation `refreshTick += 1` calls replaced with targeted loaders

- `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`:
  - Same pattern — removed `connectEventStream`/`refreshTick`
  - Streaming `$effect` dispatches to `loadModeration()`, `loadSpeakers()`, `loadAttendees()`, `loadAgenda()`, `loadVotes()`
  - Added five silent targeted loader functions
  - All 16 `refreshTick += 1` post-mutation calls replaced with appropriate loaders (Python script)

## Cleanup Completed

The remaining raw SSE plumbing has been removed:

- deleted `internal/api/http/realtime.go`
- deleted `web/src/lib/utils/sse.ts`
- removed raw SSE route registrations from `cmd/serve.go`, `e2e/helpers_test.go`, `e2e/oauth_helpers_test.go`, and `internal/api/connect/testserver_test.go`
- removed `events_url` from both meetings and moderation protos, regenerated Go + TypeScript clients, and stopped populating the field in service responses
- updated `TestRealtime_MeetingEvents_PublishesSignupInvalidation` to assert against `MeetingService.SubscribeMeetingEvents` and to cleanly close the stream so `httptest.Server` teardown does not hang

Verification completed successfully:

```bash
nix develop . --command sh -lc 'PATH="$PATH:$PWD/web/node_modules/.bin" buf generate --template buf.gen.yaml'
nix develop . --command go build ./...
nix develop . --command go test ./...
nix develop . --command sh -lc 'cd web && npm run check'
nix develop . --command sh -lc 'cd web && npm run build'
PLAYWRIGHT_DRIVER_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0 \
PLAYWRIGHT_BROWSERS_PATH=/nix/store/y53pinyaz63p6hs8acbgjnn585wnnr08-playwright-browsers-chromium \
PLAYWRIGHT_NODEJS_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0/node \
  nix develop . --command go test -count=1 -tags=e2e -timeout=300s ./e2e/...
```

## IMMEDIATE NEXT STEPS

No known cleanup blockers remain for Phase 6b. The next agent can move on to whatever follows legacy-removal completion, or prepare the commit using the message below.

Commit message suggestion:
```
feat: replace raw SSE endpoint with Connect server-streaming RPC

Replaces the hybrid GET /api/realtime/meetings/{meetingId}/events raw
SSE endpoint + per-event typed-RPC refetch pattern with a single typed
MeetingService.SubscribeMeetingEvents Connect server-streaming RPC.

The wire protocol is identical (SSE over HTTP/1.1) but the contract is
now fully typed — no raw EventSource, no manual event-name dispatch, no
separate refetch round trip. The stream sends a MeetingEventKind enum
value; the SPA dispatches to the specific targeted loader for that view.

- proto: SubscribeMeetingEvents RPC + MeetingEventKind enum + MeetingEvent
- internal/api/connect/meeting_handler.go: streaming handler subscribes
  to broker, maps event names to typed kinds, streams to client
- web live + moderate pages: Connect streaming $effect with targeted
  per-kind loaders (loadSpeakers, loadVotes, loadAgenda, etc.)
- removed internal/api/http/realtime.go (raw SSE endpoint)
- removed web/src/lib/utils/sse.ts (raw EventSource wrapper)
```

## Phase 6 Decoupling Status

- `serve.go` — fully decoupled from `internal/handlers`
- `serve_common.go` — no `internal/handlers` import; no `handler` field on `serveRuntime`
- `serve_legacy.go` — only remaining consumer of `internal/handlers`; constructs `handlers.Handler` inline
- `e2e/` helpers — use `apihttp.OAuthHandler` directly; no `handlers` import

Legacy packages still in repo (consumed only by `serve-legacy`):
- `internal/handlers/` — ~6600 lines
- `internal/templates/` — 57 files (Templ components + generated)
- `internal/routes/` — generated router + URL builders
- `tools/routing/` — route code generator
- `routes.yaml` — 1359 lines

## Most Recent Commits

- `3ff0de6` `docs: update HANDOFF.md with Phase 6 decoupling status`
- `98e9a8b` `refactor(cmd): move handlers.Handler creation into serve_legacy.go`
- `1b0613c` `feat: phase 6 legacy removal — SPA parity, new REST endpoints, OAuth extraction`
- `a107c30` `feat: continuing spa migration`

**Current HEAD**: `3ff0de6` — Phase 6b changes are uncommitted (dirty working tree)

## Suggested Verification After Each Change

```bash
nix develop . --command go test ./...
nix develop . --command sh -lc 'cd web && npm run check'
nix develop . --command sh -lc 'cd web && npm run build'
PLAYWRIGHT_DRIVER_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0 \
PLAYWRIGHT_BROWSERS_PATH=/nix/store/y53pinyaz63p6hs8acbgjnn585wnnr08-playwright-browsers-chromium \
PLAYWRIGHT_NODEJS_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0/node \
  nix develop . --command go test -count=1 -tags=e2e -timeout=300s ./e2e/...
```

## Environment Notes

- `node`, `go`, and `buf` are available in the Nix devshell
- approved command prefix: `nix develop . --command`
- **Svelte files use tabs** — use Python scripts with raw tab characters for precise edits if the Edit tool fails to match
- After any `.svelte` change, `npm run build` MUST be run before E2E tests can see the changes
- commit messages in this repo should be detailed: short subject, body paragraph, flat bullet list
- There is a devshell bootstrap quirk: `/tmp/nix-shell.XXXXXX: line 2172: $'\r': command not found` — this does not block commands

## Important Files

- [`internal/api/connect/meeting_handler.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/connect/meeting_handler.go) — streaming handler (NEW this session)
- [`internal/api/http/realtime.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/http/realtime.go) — raw SSE endpoint (TO BE DELETED)
- [`web/src/lib/utils/sse.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/utils/sse.ts) — raw EventSource wrapper (TO BE DELETED)
- [`proto/conference/meetings/v1/meetings.proto`](/mnt/c/Users/Patric/Projects/conference-tool/proto/conference/meetings/v1/meetings.proto) — has new streaming RPC
- [`web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/+page.svelte) — live page (updated)
- [`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/moderate/+page.svelte) — moderate page (updated)
