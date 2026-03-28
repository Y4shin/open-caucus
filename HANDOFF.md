# Agent Handoff

## Purpose

This file captures the current frontend rewrite state so another agent can continue Phase 6 legacy-removal work without re-discovering the recent Phase 5 completion work. It is being updated incrementally during active work so another agent can pick up mid-slice if quota runs out.

Date: 2026-03-28
Repo state at handoff update: dirty working tree — full E2E suite passes

## Current Rewrite Status

Source of truth:

- [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md)
- [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md)

Plan snapshot:

- Current phase: `Phase 6 - Legacy Removal`
- Phase 0–5: completed
- Phase 6: in progress

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
- full agenda management:
  - top-level create/activate/deactivate/reorder/delete
  - sub-point creation with parent selection
  - agenda import dialog with diff/accept/deny
- full voting parity:
  - moderator votes panel (draft, open, close, archive)
  - attendee open and secret-ballot submission
- admin/account management parity
- docs/public verification parity
- attachments and file-serving flows

## Latest Phase 6 Direction

The current migration direction is:

- keep `serve` as SPA + typed API only
- move the legacy HTMX/Templ app to a separate `serve-legacy` subcommand
- migrate browser E2E coverage to the new SPA implementation now, even while `serve-legacy` still exists for manual comparison/debugging

## What Landed In This Session

### Phase 6 Part 1: Feature parity + SPA committee management

Backend/API fixes:

- `internal/services/attendees/service.go`:
  - `SelfSignup`: removed the `!meeting.SignupOpen` gate — committee members may always self-signup; `signupOpen` only gates guest joins
  - `AttendeeLogin`: changed error message from "invalid attendee secret" to "Invalid access code"
  - Fixed `declared and not used: meeting` compile error (now uses `_` for GetMeetingByID result)

- `internal/services/speakers/service.go`:
  - `buildQueueView`: only filters WITHDRAWN (not DONE) — legacy parity where chairs see full history

- `internal/api/connect/speakers_handler_test.go`:
  - Updated `TestSpeakerService_SetSpeakerSpeaking_ThenDone` to expect DONE speaker remains in queue (1 row, DONE state)

- `internal/api/connect/attendee_handler_test.go`:
  - Renamed `TestAttendeeService_SelfSignup_SignupClosed` → `TestAttendeeService_SelfSignup_SignupClosed_MemberCanAlwaysSignup`
  - Now expects success (not error) when signup is closed but caller is a committee member

- `internal/api/http/committee_meetings.go` (NEW):
  - `NewCommitteeMeetingCreateHandler` — `POST /committee/{slug}/meetings`
  - `NewCommitteeMeetingDeleteHandler` — `DELETE /committee/{slug}/meetings/{meetingId}`
  - `NewCommitteeMeetingActivateHandler` — `POST /committee/{slug}/meetings/{meetingId}/active` (toggle)
  - All require chairperson/admin role; use session-based auth

- `cmd/serve.go` (`newAPIMux`): registered the 3 new REST endpoints

- `e2e/helpers_test.go`: registered the 3 new REST endpoints in the test server

Frontend fixes (various Svelte pages, tab-indented):

- `web/src/routes/committee/[committee]/+page.svelte` — full rewrite:
  - Chairperson/admin view: `#meeting-list-container` with create form (`data-testid="committee-create-form"`, `input[name=name]`), per-meeting rows (`data-testid="committee-meeting-row"`), active checkbox (`data-testid="committee-toggle-active"`), delete button (`data-testid="committee-delete-meeting"`)
  - Member view: active meeting card (`data-testid="committee-active-meeting-card"`, `data-testid="committee-active-meeting-name"`, `data-testid="committee-join-active-meeting"`)
  - Optimistic checkbox toggle for immediate visual feedback before server response

- `web/src/routes/committee/[committee]/meeting/[meetingId]/agenda-point/[agendaPointId]/tools/+page.svelte`:
  - Added `id="attachment-label-{agendaPointId}"` and `id="attachment-file-{agendaPointId}"` to form inputs
  - Added `id="attachment-item-{attachment.attachmentId}"` to each attachment container
  - Changed attachment label from `<span>` to `<a class="... link link-hover" href={downloadUrl}>`
  - Added `<h4>` heading between Upload and Attachments cards

- `web/src/routes/committee/[committee]/meeting/[meetingId]/attendee-login/+page.svelte`:
  - Wrapped error display in `<div id="app-notification-target">`

- `web/src/routes/committee/[committee]/meeting/[meetingId]/join/+page.svelte`:
  - Added `name="full_name"` and `name="meeting_secret"` to form inputs
  - Wrapped actionError display in `<div id="app-notification-target">`

E2E test fixes:

- `e2e/committee_test.go`: changed `form[hx-post*='/delete'] button[type=submit]` → `button[data-testid='committee-delete-meeting']`

### Phase 6 Part 2: OAuth handler extraction — decouple `serve` from `internal/handlers`

- `internal/api/http/oauth.go` (NEW):
  - `OAuthHandler` struct with `OAuthService`, `Repository`, `SessionManager`, `AuthConfig` fields
  - `NewOAuthStartHandler(h *OAuthHandler) http.Handler` — initiates OAuth/OIDC login flow
  - `NewOAuthCallbackHandler(h *OAuthHandler) http.Handler` — processes callback, resolves/provisions account, syncs admin+committees, creates session
  - Private helpers: `oauthEnabled`, `resolveAccount`, `validateRequiredGroups`, `syncAdmin`, `syncCommittees`, `upsertIdentity`, `oauthGroupContains`, `oauthRoleRank`, `oauthStrPtr`
  - This is a standalone extraction; the legacy `internal/handlers/oauth.go` remains for `serve-legacy`

- `cmd/serve.go`:
  - Creates `oauthH := &apihttp.OAuthHandler{...}` from runtime deps
  - Routes `/oauth/start` and `/oauth/callback` now use `apihttp.NewOAuthStartHandler(oauthH)` and `apihttp.NewOAuthCallbackHandler(oauthH)`
  - **`serve.go` no longer imports or references `internal/handlers` at all**

- `e2e/helpers_test.go`:
  - Replaced `h := &handlers.Handler{...}` with `oauthH := &apihttp.OAuthHandler{...}`
  - Routes updated to use `apihttp.NewOAuthStartHandler(oauthH)` and `apihttp.NewOAuthCallbackHandler(oauthH)`
  - Removed `"github.com/Y4shin/conference-tool/internal/handlers"` import

- `e2e/oauth_helpers_test.go`:
  - Same OAuth migration: uses `apihttp.OAuthHandler` instead of `handlers.Handler`
  - Removed `handlers` import

**Decoupling achieved**: `internal/handlers` is now only referenced from `cmd/serve_common.go` (for `serve-legacy`). The `serve` command is fully independent.

## Current Verification Snapshot

All passing (verified after OAuth extraction):

- `nix develop . --command go build ./...` — clean build
- `nix develop . --command go test ./...` — all Go unit/integration tests pass
- Full E2E suite: all tests pass in ~14s

## Immediate Next Steps

The full E2E suite passes and `serve` is decoupled from `internal/handlers`. Next work:

1. Commit the current batch of Phase 6 changes.
2. Continue Phase 6 legacy removal: look at what remains in `serve_common.go` that only `serve-legacy` needs (the `handler` field on `serveRuntime`).
3. Consider extracting or removing the `rt.handler` field from the shared `serveRuntime` struct so it is `serve-legacy`-only.
4. Identify any remaining legacy routes/handlers in `internal/handlers` that are only used by `serve-legacy`.
5. After each change: run `go test ./...` and the full E2E suite.

## Suggested Verification After Each Change

```bash
nix develop . --command go test ./...
nix develop . --command sh -lc 'cd web && npm run check'
nix develop . --command sh -lc 'cd web && npm run build'
PLAYWRIGHT_DRIVER_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0 \
PLAYWRIGHT_BROWSERS_PATH=/nix/store/y53pinyaz63p6hs8acbgjnn585wnnr08-playwright-browsers-chromium \
PLAYWRIGHT_NODEJS_PATH=/nix/store/hm3dzl8q4cj08sxd64ha49npppkiwa5i-playwright-driver-1.52.0/node \
  nix develop . --command go test -count=1 -v -tags=e2e -timeout=300s ./e2e/...
```

## Important Files

Primary SPA surfaces:

- [`web/src/routes/committee/[committee]/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/+page.svelte) — committee home with meeting management
- [`web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte) — live meeting page
- [`web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte) — moderation workspace

REST endpoints (non-Connect):

- [`internal/api/http/committee_meetings.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/http/committee_meetings.go) — meeting create/delete/toggle-active
- [`internal/api/http/oauth.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/http/oauth.go) — OAuth start/callback handlers (extracted from internal/handlers)
- [`internal/api/http/attachments.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/api/http/attachments.go) — file upload

Typed client entrypoints:

- [`web/src/lib/api/index.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/api/index.ts)
- [`web/src/lib/api/services.ts`](/mnt/c/Users/Patric/Projects/conference-tool/web/src/lib/api/services.ts)

Backend service layer:

- [`internal/services/attendees/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/attendees/service.go)
- [`internal/services/speakers/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/speakers/service.go)
- [`internal/services/committees/service.go`](/mnt/c/Users/Patric/Projects/conference-tool/internal/services/committees/service.go)

## Environment Notes

- `node`, `go`, and `buf` are available in the Nix devshell
- approved command prefix already exists for: `nix develop . --command`
- **Svelte files use tabs** — use Python scripts with raw tab characters for precise edits if the Edit tool fails to match
- After any `.svelte` change, `npm run build` MUST be run before E2E tests can see the changes
- commit messages in this repo should be detailed: short subject, body paragraph, flat bullet list
- There is a devshell bootstrap quirk: `/tmp/nix-shell.XXXXXX: line 2172: $'\r': command not found` — this does not block commands

## Most Recent Commits

Recent rewrite commits, newest first (uncommitted work is above `a107c30`):

- `a107c30` `feat: continuing spa migration` ← current HEAD (Phase 6 work not yet committed)
- `d4cd535` `feat(web): add typed voting workflow to moderate and live pages`
- `0972109` `feat(web): add agenda management to moderation workspace`
- `467d64f` `feat(web): add moderation attendee search for speakers`
- `3f87594` `feat(web): add speaker queue controls to live and moderate`
- `fe2a8a0` `feat(web): port attendee join and login flows`

## Pickup Advice For The Next Agent

If you are continuing immediately, start here:

1. Re-read [`frontend-rewrite-plan.md`](/mnt/c/Users/Patric/Projects/conference-tool/frontend-rewrite-plan.md) and keep it as the source of truth.
2. Re-read [`e2e-api-mapping-matrix.md`](/mnt/c/Users/Patric/Projects/conference-tool/e2e-api-mapping-matrix.md) before removing or replacing legacy workflows.
3. The full E2E suite is green. Any next change that breaks it must be fixed before moving on.
4. Keep verification tight after each slice:
   - `npm run check`
   - `npm run build`
   - `go test ./...`
   - Full E2E suite
