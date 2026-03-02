---
title-en: Committee Dashboard and Meeting Lifecycle
title-de: Gremiumsseite und Meeting-Lebenszyklus
---

# Gremiumsseite und Meeting-Lebenszyklus

![Gremiumsseite Chair](../../assets/captures/app-committee-dashboard-chair.en.light.desktop.png)

## Lebenszyklus-Aktionen

- Meeting erstellen: `POST /committee/{slug}/meeting/create`
- Meeting aktivieren/deaktivieren: `POST /committee/{slug}/meeting/{meeting_id}/activate`
- Signup-Status umschalten: `POST /committee/{slug}/meeting/{meeting_id}/signup-open-toggle`
- Meeting löschen: `POST /committee/{slug}/meeting/{meeting_id}/delete`

## Praktische Reihenfolge

1. Meeting mit klarem Namen und optionaler Beschreibung anlegen.
2. Genau ein aktives Meeting für schnellen Beitritt setzen.
3. Signup-Status mit der tatsächlichen Praxis synchron halten.
