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

*(planned)*

### 2.1 SpeakerList / AgendaList components
- Large meeting page (906 lines) and moderate page (2614 lines) have domain-specific repeated patterns.
- To be extracted in a follow-up pass.

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

---

## Notes for Next Agent

- The project uses **Svelte 5** (snippets replace slots — use `{#snippet name()}...{/snippet}` and `{@render name()}`)
- The project uses **DaisyUI** + Tailwind for styling
- All user-facing text uses **Paraglide** i18n via `import * as m from '$lib/paraglide/messages'`
- Run e2e tests with `task test:e2e` (requires Playwright browsers via `task playwright:install`)
- Commit regularly with descriptive messages
- The `moderate/+page.svelte` (2614 lines) is the biggest opportunity but also most complex — approach carefully
- AppCard accepts `title?: string` and `class?: string` props; the admin pages need `class="mb-4 bg-base-100 shadow-sm"` to get the right look
