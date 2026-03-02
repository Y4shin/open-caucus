---
title-en: Agenda Management and Import
title-de: Tagesordnung und Import
---

# Tagesordnung und Import

## Tagesordnungsrouten

- Punkt erstellen: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/create`
- Punkt löschen: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/delete`
- Reihenfolge ändern: `.../move-up` und `.../move-down`
- Punkt aktivieren: `.../activate`

## Importablauf

- Vorschlag extrahieren: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/extract`
- Diff prüfen: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/diff`
- Änderungen anwenden: `POST /committee/{slug}/meeting/{meeting_id}/agenda/import/apply`

## Empfehlungen

- Struktur vor dem Anwenden prüfen.
- Aktiven Tagesordnungspunkt mit der Debatte synchron halten.
