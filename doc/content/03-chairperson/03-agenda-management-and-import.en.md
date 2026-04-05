---
title-en: Agenda Management and Import
title-de: Tagesordnung und Import
---

# Agenda Management and Import

Use this page to maintain agenda points during the meeting and import a prepared agenda safely.

## Who this is for

Chairpersons who keep the agenda organized and aligned with the current discussion.

## Before you start

1. Open the meeting moderation page and switch to the **Agenda** tab.
2. Decide whether you need manual edits, import-based updates, or both.
3. If importing, prepare your agenda as pasted text (for example text with `#` headings or numbered lines).

## Layout on desktop vs mobile

- Desktop:
  You can keep the agenda controls visible while also viewing other moderation areas.
- Mobile:
  Work the **Agenda** tab first, then scroll to other moderation sections as needed.

## Edit Agenda

Use this for normal day-to-day agenda maintenance during a meeting.

### Step-by-step

1. Add agenda points:
   - In **Add Agenda Point**, enter a title.
   - Optionally choose a parent in **Parent (optional)**.
   - Click **Add**.
2. Reorder points:
   - Use **Move up** and **Move down** on agenda cards.
3. Activate the current discussion point:
   - Click **Activate agenda point** on the correct item.
   - The entry time is recorded and shown next to each point in the sidebar. When you switch to another point, the duration is also recorded.
4. Open point-specific actions:
   - Click **Open tools** on the agenda card.
5. Remove obsolete points:
   - Click **Delete agenda point** and confirm.

### What to confirm after edits

- The active point matches the real discussion.
- Parent/child structure is still correct.
- On mobile, re-check after scrolling back to the top of the list.

## Import Agenda (Overview)

Use this when you have a prepared agenda text and want controlled bulk updates.

1. Click **Import** to open **Import Agenda**.
2. In **Source**, paste text or upload a file, then click **Extract Agenda**.
3. In **Correction**, review detected lines and click lines to set `Ignore`, `Heading` (main point), or `Subheading` (subpoint).
4. Click **Generate Diff**.
5. In **Diff** (change preview), review all changes carefully.
6. Click **Accept** to apply or **Deny** to cancel.
7. After applying, confirm the active point is still correct.

## Import Agenda (How It Works)

### Which Text Formats Work

1. Text with `#` and `##` headings:
   - Lines starting with `#` are treated as main points.
   - Lines starting with `##` are treated as subpoints.
   - In some agenda styles, the first `#` line is just a title; in that case the importer focuses on the lower heading levels.
2. Numbered text:
   - The importer understands common formats like `1`, `1.1`, or `TOP1`.
   - Main numbers become agenda points, and nested numbers become subpoints.
3. Indented text:
   - Less-indented lines become main points.
   - More-indented lines become subpoints.
4. If a line does not fit the chosen structure, it can be set to `Ignore` in the correction step.

### Correction Options

- `Ignore`: skip this line.
- `Heading`: use as a main agenda point.
- `Subheading`: use as a child point under the latest main point.
- A `Subheading` line must come after a `Heading` line.

### Change Preview Logic

1. The importer builds a proposed agenda from the corrected lines.
2. It compares that proposal with your current agenda:
   - First it tries exact title matches.
   - If that fails, it tries likely matches based on similar wording and position.
3. Each row in the change preview is marked as one of:
   - `insert`: new point will be added
   - `delete`: existing point will be removed
   - `move`: point will stay but change position/parent
   - `rename`: point keeps position but title changes
   - `unchanged`: no change
4. This preview is designed to show all changes clearly before you apply them.

### Safety Check Before Apply

- The app checks whether the agenda changed while you were reviewing the preview.
- If it changed, you will see a warning and must review the updated preview again before applying.

## If something goes wrong

- Import says source is empty/too large/unparseable:
  Clean the source text and retry with a smaller, structured input.
- Correction step fails:
  Ensure at least one line remains `Heading` (main point), and avoid `Subheading` (subpoint) before any main point.
- Diff warns agenda changed while reviewing:
  Re-check the updated change preview before accepting.
- Imported result looks wrong:
  Use **Deny**, adjust source/corrections, and generate the preview again.

## Agenda Routes

Use this control map when Moderate help opens this page at `agenda-routes`:

- **Agenda** tab: day-to-day point management.
- **Add Agenda Point**: manual create flow.
- **Import Agenda**: `Source` -> `Correction` -> `Diff` import steps.
- **Activate agenda point**: sets the point currently being discussed.

## What happens next

Continue with [Attendees Signup and Recovery](/docs/03-chairperson/04-attendees-signup-and-recovery) and [Speakers Moderator and Quotation](/docs/03-chairperson/05-speakers-moderator-and-quotation) to run the live meeting.
