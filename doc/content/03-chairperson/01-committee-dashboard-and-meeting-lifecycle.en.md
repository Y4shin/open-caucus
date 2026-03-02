---
title-en: Committee Dashboard and Meeting Lifecycle
title-de: Gremiumsseite und Meeting-Lebenszyklus
---

# Committee Dashboard and Meeting Lifecycle

![Chair committee dashboard](../../assets/captures/app-committee-dashboard-chair.en.light.desktop.png)

## Lifecycle actions

- Create meeting: `POST /committee/{slug}/meeting/create`
- Activate/deactivate meeting: `POST /committee/{slug}/meeting/{meeting_id}/activate`
- Toggle signup-open: `POST /committee/{slug}/meeting/{meeting_id}/signup-open-toggle`
- Delete meeting: `POST /committee/{slug}/meeting/{meeting_id}/delete`

## Practical sequence

1. Create meeting with a clear name and optional description.
2. Activate exactly one meeting for member quick join.
3. Keep signup state aligned with actual in-room policy.
