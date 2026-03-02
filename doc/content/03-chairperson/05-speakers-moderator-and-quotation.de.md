---
title-en: Speakers Moderator and Quotation
title-de: Redeliste Moderation und Quotierung
---

# Redeliste, Moderation und Quotierung

## Redelistenrouten

- Redebeitrag hinzufügen: `POST /committee/{slug}/meeting/{meeting_id}/speaker/add`
- Entfernen: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/remove`
- Starten: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/start`
- Beenden: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/end`
- Zurückziehen: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/withdraw`
- Priorität umschalten: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/priority`

## Quotierung und Moderationseinstellungen

- Meeting-Standardquotierung: `POST /committee/{slug}/meeting/{meeting_id}/quotation`
- Meeting-Moderator setzen: `POST /committee/{slug}/meeting/{meeting_id}/moderator`
- Quotierung pro Tagesordnungspunkt: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/quotation`
- Moderator pro Tagesordnungspunkt: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/moderator`
