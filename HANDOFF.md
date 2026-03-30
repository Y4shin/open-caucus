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

`A00` is complete locally and ready to be checkpointed as the baseline commit.

Included changes:

- fixed the extended parity tests in `e2e/ui_parity_extended_test.go`
- updated the committee, live, join, and moderate SPA routes to close the known legacy-parity gaps
- added `ui-parity-expansion-plan.md`
- added the atomic-task execution model and queue to the parity plan
- refreshed this handoff for commit-per-task execution

Verification already completed successfully from this working tree state:

- `nix develop -c bash -lc 'cd web && npm run build'`
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"`
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/...`

## Baseline Status

The worktree is currently dirty only because it contains the verified `A00` baseline changes that are about to be committed.

After the baseline commit lands, the next task should be `A01`.

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

Start with `A01`.

Definition of done for `A01`:

- add a small helper for post-action fragment parity comparisons
- keep the task limited to helper extraction plus any minimal adoption needed to prove it
- verify with a focused parity test, then the full parity suite, then the full E2E suite
- update this handoff to point at the next atomic task after the helper lands

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
