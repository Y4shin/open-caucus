---
title-en: Permission Model and Common 403 Cases
title-de: Berechtigungsmodell und typische 403-Fälle
---

# Permission Model and Common 403 Cases

## Middleware gates

- `auth`, `session`, `admin_required`
- `committee_access`, `meeting_access`, `moderate_access`

## Frequent 403 causes

- Non-admin access to `/admin` routes.
- Non-committee users opening `/committee/{slug}` resources.
- Non-chair attendees opening `/committee/{slug}/meeting/{meeting_id}/moderate`.
- Missing meeting-attendee relationship for live actions.

## Operational check

When debugging permissions, validate role, committee membership, attendee linkage, and current session identity in this order.
