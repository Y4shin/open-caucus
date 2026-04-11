# bits-ui Migration Plan

Migrate interactive DaisyUI/HTML components to bits-ui for proper accessibility (ARIA roles, keyboard navigation, focus management). Keep DaisyUI classes for styling.

## 1. Tooltips (15+ instances, 5 files)

**Why:** DaisyUI tooltips are CSS-only — no `role="tooltip"`, no keyboard trigger, no delay control.

- [x] Create shared `AppTooltip.svelte` wrapper component
- [x] `web/src/lib/components/ui/SpeakerBadges.svelte` — 4 tooltip spans
- [x] `web/src/lib/components/ui/AttendeeRow.svelte` — 4 tooltips (badges + action buttons)
- [x] `web/src/lib/components/ui/AgendaPointCard.svelte` — 5 icon button tooltips
- [x] `web/src/routes/committee/[committee]/meeting/[meetingId]/+page.svelte` — 4 document/speaker button tooltips
- [x] Removed `data-tip` / DaisyUI tooltip classes from migrated elements

## 2. Selects (10 instances, 5 files)

**Why:** Native `<select>` has no keyboard search, limited styling, poor mobile UX. bits-ui Select adds typeahead, custom rendering, proper ARIA listbox pattern.

- [x] Create shared `AppSelect.svelte` wrapper component
- [x] `web/src/routes/admin/committee/[slug]/+page.svelte` — 4 selects (account, role x2, oauth role)
- [x] `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/AgendaSection.svelte` — 1 select (parent point)
- [ ] `web/src/lib/components/ui/VoteCard.svelte` — 1 select (visibility) — **kept native** (FormData submission)
- [ ] `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/VotesPanelSection.svelte` — 1 select (visibility) — **kept native** (FormData submission)
- [x] `web/src/routes/committee/[committee]/meeting/[meetingId]/moderate/+page.svelte` — 3 selects (quotation x2, moderator)

## 3. Accordions (4 instances, 3 files) — **Skipped**

**Why skipped:** All instances are standalone `<details>` collapsibles, not grouped accordion panels. Native `<details>` already provides keyboard support (Enter/Space), built-in ARIA semantics, and screen reader compatibility. bits-ui Accordion/Collapsible adds no meaningful a11y improvement for standalone collapsibles.

- DocsOverlay — nested tree structure, better suited to a future TreeView component
- VoteCard — standalone collapsible, native `<details>` is appropriate
- VotesPanelSection — standalone collapsible, native `<details>` is appropriate

## 4. Dropdown Menu (1 instance, 1 file)

**Why:** Manual toggle with no focus trap, no keyboard nav, no auto-close on outside click.

- [x] `web/src/routes/+layout.svelte` — mobile menu dropdown → bits-ui Popover (auto-close on outside click, focus trap, keyboard dismiss)

## Not migrating

- **Dialogs (8 instances):** Native `<dialog>` already accessible, has focus trap and Escape-to-close.
- **Toggles (6 instances):** Native checkbox with DaisyUI styling already accessible.
