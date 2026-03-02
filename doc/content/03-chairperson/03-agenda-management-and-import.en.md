---
title-en: Agenda Management and Import
title-de: Tagesordnung und Import
---

# Agenda Management and Import

## Agenda routes

- Create point: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/create`
- Delete point: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/delete`
- Move up/down: `.../move-up` and `.../move-down`
- Activate point: `.../activate`

## Import flow

- extract proposal: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/extract`
- compare diff: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/diff`
- apply changes: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/apply`

## Recommendations

- Validate imported structure before apply.
- Keep active agenda point aligned with live discussion.
