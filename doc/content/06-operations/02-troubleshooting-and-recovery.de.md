---
title-en: Troubleshooting and Recovery
title-de: Troubleshooting und Wiederherstellung
---

# Troubleshooting und Wiederherstellung

## Typische Störungen

- verlorene Zugangscodes von Teilnehmenden
- falsches aktives Meeting
- veralteter Browserzustand nach Rollen-/Sessionwechsel
- unvollständiger Abstimmungsablauf (Status counting)

## Recovery-Maßnahmen

- Recovery-Link/QR: `/committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/recovery`
- aktives Meeting im Gremiumsdashboard korrigieren
- HTMX-Panels mit `Refresh` vor kritischen Aktionen neu laden
- abgeschlossene Abstimmungen archivieren

## Integritätschecks für Dokumentation

- jedes Verzeichnis mit `index` in EN+DE
- jede Seite mit `title-en` und `title-de`
- Docs-Tests vor Deployment ausführen
