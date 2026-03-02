---
title-en: Attachments and Current Document
title-de: Anhänge und aktuelles Dokument
---

# Attachments and Current Document

![Agenda tools attachments](../../assets/captures/app-agenda-tools-attachments.en.light.desktop.png)

## Attachment routes

- Tools page: `GET /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/tools`
- Upload attachment: `POST .../attachment/create`
- Delete attachment: `POST .../attachment/{attachment_id}/delete`
- Set current document: `POST .../attachment/{attachment_id}/set-current`
- Clear current document: `POST .../clear-current`

## Live document delivery

- Current document stream endpoint: `GET /committee/{slug}/meeting/{meeting_id}/current-doc`

The current document is attendee-visible and should always reflect the active discussion item.
