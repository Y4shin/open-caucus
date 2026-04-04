# Frontend Component Refactoring Progress

## Goal

Extract reusable Svelte components from large, inline-heavy pages to reduce duplication and improve maintainability. All changes must pass E2E tests.

## Status: In Progress

---

## Completed

*(none yet)*

---

## Phase 1: Cross-page UI primitives

### 1.1 PaginationNav — DONE
- **Component**: `web/src/lib/components/ui/PaginationNav.svelte`
- **Applied to**: `admin/+page.svelte`, `admin/accounts/+page.svelte`, `admin/committee/[slug]/+page.svelte`, `committee/[committee]/+page.svelte`
- **Pattern extracted**: The static disabled pagination control (prev / 1 / next) was copy-pasted verbatim across 4 files.

### 1.2 DataTable — DONE
- **Component**: `web/src/lib/components/ui/DataTable.svelte`
- **Applied to**: `admin/+page.svelte`, `admin/accounts/+page.svelte`, `admin/committee/[slug]/+page.svelte`
- **Pattern extracted**: `<table class="data-table table table-zebra w-full">` with `header` and `body` Svelte 5 snippets.

### 1.3 AppCard adoption in admin pages — DONE
- **Component**: `web/src/lib/components/ui/AppCard.svelte` (already existed, was unused)
- **Applied to**: `admin/+page.svelte`, `admin/accounts/+page.svelte`, `admin/committee/[slug]/+page.svelte`
- **Pattern extracted**: Inline `<section class="panel card bg-base-100 border border-base-300 shadow-sm rounded-box p-4 mb-4">` replaced with `<AppCard>`.

---

## Phase 2: Meeting & Moderate pages

### 2.1 SpeakerBadges component — DONE
- **Component**: `web/src/lib/components/ui/SpeakerBadges.svelte`
- **Applied to**: `meeting/+page.svelte` (2 occurrences, also fixed a redundant double `{#if}`), `moderate/+page.svelte` (1 occurrence)
- **Pattern extracted**: The badge row for ROPM, quoted, firstSpeaker, priority, and "you" was copy-pasted verbatim.
- **Props**: `speakerType`, `quoted`, `firstSpeaker`, `priority`, `mine` (optional, default false)

### 2.2 Vote badge utilities — DONE
- **Module**: `web/src/lib/utils/votes.ts`
- **Exports**: `voteStateBadgeClass(state)`, `voteVisibilityBadgeClass(visibility)`
- **Applied to**: both `meeting/+page.svelte` and `moderate/+page.svelte` — removed local duplicates.

---

## What Remains (Phase 3 ideas)

The moderate page is still 2600+ lines. Further opportunities if a future agent wants to continue:
- Extract the **AgendaPointList** section (lines ~1460–1650) into a component
- Extract the **VoteCard** per-vote accordion (lines ~1800–2100) into a component
- Extract the **AttendeeRow** (the large attendee list row in manage tab, lines ~2130–2185) into a component
- The `join/+page.svelte` (211 lines) has some inline form patterns that could use `AppCard`

---

## File Line Counts (before refactoring)

| File | Lines |
|------|-------|
| moderate/+page.svelte | 2614 |
| meeting/+page.svelte | 906 |
| admin/committee/[slug]/+page.svelte | 375 |
| committee/[committee]/+page.svelte | 302 |
| admin/+page.svelte | 190 |
| admin/accounts/+page.svelte | 186 |
| join/+page.svelte | 211 |
| agenda-point/tools/+page.svelte | 248 |

---

## Components Inventory

| Component | File | Status |
|-----------|------|--------|
| AppAlert | `ui/AppAlert.svelte` | Pre-existing |
| AppCard | `ui/AppCard.svelte` | Pre-existing, now used |
| AppSpinner | `ui/AppSpinner.svelte` | Pre-existing |
| LegacyIcon | `ui/LegacyIcon.svelte` | Pre-existing |
| PaginationNav | `ui/PaginationNav.svelte` | Phase 1 |
| DataTable | `ui/DataTable.svelte` | Phase 1 |
| SpeakerBadges | `ui/SpeakerBadges.svelte` | Phase 2 |
| votes utils | `utils/votes.ts` | Phase 2 |

---

## Notes for Next Agent

- The project uses **Svelte 5** (snippets replace slots — use `{#snippet name()}...{/snippet}` and `{@render name()}`)
- The project uses **DaisyUI** + Tailwind for styling
- All user-facing text uses **Paraglide** i18n via `import * as m from '$lib/paraglide/messages'`
- Run e2e tests with `task test:e2e` (requires Playwright browsers via `task playwright:install`)
- Commit regularly with descriptive messages
- The `moderate/+page.svelte` (2614 lines) is the biggest opportunity but also most complex — approach carefully
- AppCard accepts `title?: string` and `class?: string` props; the admin pages need `class="mb-4 bg-base-100 shadow-sm"` to get the right look
