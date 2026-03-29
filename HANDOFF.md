# Handoff

## Current Goal

Finish the SPA migration so the active `serve` path does not serve any legacy HTMX/Templ HTML. The legacy stack should only remain available as a comparison target for parity tests.

## What Was Completed Before This Checkpoint

- Removed the raw SSE path and moved meeting streaming to Connect.
- Moved docs page/search to Connect.
- Moved several remaining `/api` JSON routes to Connect.
- Added a `render-template` command for rendering legacy templ components from JSON input.
- Added and expanded UI parity E2E coverage between the legacy server and the SPA.
- Ported a large portion of the SPA UI closer to the legacy appearance.
- Fixed several SPA/live/moderation behavior bugs:
  - live page vote panel hydration/runtime loop
  - speaker queue re-add behavior for `DONE` speakers
  - completed speakers no longer showing `0`
  - speaking timer / active speaker indicator behavior
  - OIDC issuer/account upsert regression when switching `127.0.0.1` to `localhost`
  - dev-task / Air / Vite / Rollup / Playwright shell issues

## Important Recent Change

I started removing the legacy fallback routing from the active SPA path:

- [cmd/serve.go](/mnt/c/Users/Patric/Projects/conference-tool/cmd/serve.go)
- [e2e/helpers_test.go](/mnt/c/Users/Patric/Projects/conference-tool/e2e/helpers_test.go)

The SPA server no longer intentionally dispatches these requests to the legacy router:

- meeting-management utility POSTs
- legacy vote partial/action endpoints
- legacy agenda import/move endpoints
- legacy attendee-login POST / secret-login fallback
- legacy docs HTML endpoints like `/docs`, `/docs/search`, `/docs/oob`

The separate legacy comparison server in `newLegacyTestServer(...)` was intentionally left intact for parity tests.

## Last Confirmed Green State Before Removing SPA Fallbacks

The full E2E suite was green before the most recent fallback-removal pass.

Also confirmed green immediately before this checkpoint:

- `TestMeetingModerate_UIParityWithLegacy`
- `TestVoting_OpenVote_ModeratorAndAttendeeHappyPath_HTMX`
- `TestVoting_LivePanelUpdatesViaSSEOnVoteOpen`
- `TestVoting_Concurrent20Attendees_TallyIsCorrect`

## Key Bug Fixed Right Before Removing Fallbacks

`POST /committee/{slug}/meeting/{meetingId}/attendee-login` had regressed to `404` in the SPA server path, which broke the concurrent open-voting E2E case because the plain HTTP attendee clients never got authenticated sessions. That is why the test previously showed zero casts/ballots even though the requests were returning `200` app-shell HTML.

That route worked again right before the fallback removal, and the concurrent-voting test passed.

## Current State At This Checkpoint

- The working tree is intentionally large and includes all SPA migration/parity work to date.
- The active SPA path has just been changed to stop serving legacy HTML fallbacks.
- A fresh full E2E run was started after that removal, but I am checkpointing before finishing the next repair pass.

## Expected Next Step

Run the suite again and treat the failures as the porting checklist:

```bash
nix develop . --command bash -lc 'go test -tags=e2e -timeout=600s ./e2e/...'
```

The most likely breakages now are the SPA areas that were still quietly depending on legacy HTML endpoints:

- live votes panel in [web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/%2Bpage.svelte)
- moderation tools/votes/attendee actions in [web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/moderate/%2Bpage.svelte)
- attendee login non-JS behavior
- docs overlay/search behavior if anything still expects server-rendered docs HTML

## Files Most Likely To Need Follow-Up

- [cmd/serve.go](/mnt/c/Users/Patric/Projects/conference-tool/cmd/serve.go)
- [e2e/helpers_test.go](/mnt/c/Users/Patric/Projects/conference-tool/e2e/helpers_test.go)
- [web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/%2Bpage.svelte)
- [web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/committee/%5Bcommittee%5D/meeting/%5BmeetingId%5D/moderate/%2Bpage.svelte)
- [web/src/routes/docs/[...docPath]/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/docs/%5B...docPath%5D/%2Bpage.svelte)
- [web/src/routes/docs/search/+page.svelte](/mnt/c/Users/Patric/Projects/conference-tool/web/src/routes/docs/search/%2Bpage.svelte)

## Verification Commands Used Frequently

```bash
cd web && npm run build
nix develop . --command bash -lc 'go test -tags=e2e -timeout=600s ./e2e/...'
nix develop . --command bash -lc 'go test -tags=e2e ./e2e/... -run "Test(.*UIParityWithLegacy|Voting_.*|Moderate.*|Docs.*)" -count=1'
```
