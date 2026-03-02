---
title-en: Attachments and Current Document
title-de: Anhänge und aktuelles Dokument
---

# Anhänge und aktuelles Dokument

![Tagesordnungstools Anhänge](../../assets/captures/app-agenda-tools-attachments.en.light.desktop.png)

## Anhangsrouten

- Tools-Seite: `GET /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/tools`
- Upload: `POST .../attachment/create`
- Löschen: `POST .../attachment/{attachment_id}/delete`
- Aktuelles Dokument setzen: `POST .../attachment/{attachment_id}/set-current`
- Aktuelles Dokument löschen: `POST .../clear-current`

## Live-Dokumentausgabe

- Endpoint für aktuelles Dokument: `GET /committee/{slug}/meeting/{meeting_id}/current-doc`

Das aktuelle Dokument ist für Teilnehmende sichtbar und sollte zur laufenden Debatte passen.
