# Handoff: Fix reactive cycles in VotesPanelSection and SpeakersSection

## Status

All fixes are implemented and built. The **final E2E test run has NOT been done** after the VotesPanelSection fix — you need to run it to confirm everything passes.

## What was broken

Commit `8c9ca63` extracted `VotesPanelSection` and `SpeakersSection` from `moderate/+page.svelte`. This broke all E2E tests on the moderate page (agenda, speaker, and voting tests) with consistent 10-second Playwright timeouts.

## Root cause

Both extracted components had a **Svelte 5 reactive cycle** in their `$effect` blocks:

### SpeakersSection

```javascript
// BAD: reads speakingSinceMs (reactive $state) inside an effect,
// then writes speakingSinceMs — causing an infinite cycle
$effect(() => {
    if (speakers) syncSpeakingSince(speakers);
});

function syncSpeakingSince(spkrs) {
    const next = { ...speakingSinceMs };  // ← READ: creates dependency
    ...
    speakingSinceMs = next;               // ← WRITE: triggers the same effect
}
```

### VotesPanelSection

```javascript
// BAD: same pattern — reads voteAccordionOpen/draftVoteEditorOpen inside an
// effect, then writes to them
$effect(() => {
    if (votesPanel?.votes) syncVotePanelOpenState(votesPanel.votes);
});

function syncVotePanelOpenState(votes) {
    // reads voteAccordionOpen  ← READ: creates dependency
    // reads draftVoteEditorOpen ← READ: creates dependency
    voteAccordionOpen = nextVoteAccordionOpen;   // ← WRITE: triggers effect
    draftVoteEditorOpen = nextDraftVoteEditorOpen; // ← WRITE: triggers effect
}
```

In the old inline code these functions were called from plain async functions, so Svelte's reactive tracking didn't apply. When moved inside `$effect`, Svelte detected the read→write cycle and threw `effect_update_depth_exceeded`, crashing the component tree and making the entire moderate page unresponsive.

## Fix applied

Use `untrack()` from Svelte when reading the old state values inside the effect, breaking the dependency cycle:

### SpeakersSection — inlined the sync logic with `untrack`:
```javascript
import { untrack } from 'svelte';

$effect(() => {
    if (!speakers) return;
    const spkrs = speakers;
    const current = untrack(() => speakingSinceMs);  // ← no dependency created
    const next = { ...current };
    const activeIds = new Set(spkrs.map((s) => s.speakerId));
    for (const s of spkrs) {
        if (s.state === 'SPEAKING' && next[s.speakerId] == null) {
            next[s.speakerId] = Date.now();
        }
    }
    for (const speakerId of Object.keys(next)) {
        if (!activeIds.has(speakerId)) delete next[speakerId];
    }
    speakingSinceMs = next;
});
// syncSpeakingSince function removed (dead code)
```

### VotesPanelSection — inlined the sync logic with `untrack`:
```javascript
import { untrack } from 'svelte';

$effect(() => {
    const votes = votesPanel?.votes;
    if (!votes) return;
    const currentAccordion = untrack(() => voteAccordionOpen);   // ← no dependency
    const currentDraft = untrack(() => draftVoteEditorOpen);      // ← no dependency
    const nextVoteAccordionOpen: Record<string, boolean> = {};
    const nextDraftVoteEditorOpen: Record<string, boolean> = {};
    for (const vote of votes) {
        nextVoteAccordionOpen[vote.voteId] =
            currentAccordion[vote.voteId] ?? voteAccordionDefaultOpen(vote);
        if (vote.state === 'draft') {
            nextDraftVoteEditorOpen[vote.voteId] = currentDraft[vote.voteId] ?? false;
        }
    }
    voteAccordionOpen = nextVoteAccordionOpen;
    draftVoteEditorOpen = nextDraftVoteEditorOpen;
});
// syncVotePanelOpenState function removed (dead code)
```

Also added `onReload` prop to `SpeakersSection` (called after each speaker action and in `endCurrentSpeaker`) and pass `onReload={loadSpeakers}` from the parent. This ensures immediate UI updates after speaker actions without waiting for the 2-second polling cycle.

## What to do next

1. Build the SPA: `task build:web`
2. Run E2E tests: `task test:e2e`
3. All tests should pass. If voting tests still fail, investigate `VotesPanelSection`.
4. Commit the result (or amend this commit after confirming tests pass).

## Files changed in this session

- `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/SpeakersSection.svelte`
  - Added `onReload: () => void` prop
  - Replaced `$effect(() => syncSpeakingSince(speakers))` with inlined `untrack`-safe version
  - Removed `syncSpeakingSince` function (dead after inline)
  - Added `onReload()` call in `runSpeakerAction` (success + failure) and `endCurrentSpeaker`

- `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/VotesPanelSection.svelte`
  - Replaced `$effect(() => syncVotePanelOpenState(...))` with inlined `untrack`-safe version
  - Removed `syncVotePanelOpenState` function (dead after inline)
  - Removed `voteActionPending` state (was only written, never read)
  - Removed now-dead `getVotesPanel` call after `createVote` (the parent's polling handles refresh)

- `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte`
  - Passes `onReload={loadSpeakers}` to `<SpeakersSection>`

## Test run results before this commit

After SpeakersSection fix (VotesPanelSection fix NOT yet applied at that point):
- All `TestAgendaPoint_*` → PASS
- All `TestAgendaImport_*` → PASS
- All `TestSpeakersList_*` → PASS
- `TestVoting_*` (10 tests) → FAIL with "timeout waiting for vote X to enter open/counting controls"

The VotesPanelSection fix was applied afterward and addresses the same root cause for voting tests.
