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

`A01` is complete locally and ready to be checkpointed.

Included changes:

- added `compareFragmentAfterAction(...)` in `e2e/ui_parity_test.go`
- adopted the helper in `TestModerateSettingsTab_UIParityWithLegacy` in `e2e/ui_parity_extended_test.go`

Verification already completed successfully from this working tree state:

- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestModerateSettingsTab_UIParityWithLegacy"`
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run ".*UIParityWithLegacy"`
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestSpeakersList_OneNonDoneEntryPerType"`
- `nix develop -c go test -v -tags=e2e -timeout=600s ./e2e/... -run "TestAttachments_UploadWithoutLabel_ShowsFilename"`

Full-suite note:

- Two separate full-E2E runs ended with unrelated timeouts outside the `A01` change set:
  - `TestSpeakersList_OneNonDoneEntryPerType`
  - `TestAttachments_UploadWithoutLabel_ShowsFilename`
- Both passed immediately when rerun as focused tests.
- The parity suite was fully green after the helper change.

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

Start with `A02`.

Definition of done for `A02`:

- add live-page parity coverage for an active speaker state
- keep the change limited to one new parity scenario and any minimal helper reuse needed
- verify with a focused parity test, then the full parity suite, then the full E2E suite
- update this handoff to point at `A03` next

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
