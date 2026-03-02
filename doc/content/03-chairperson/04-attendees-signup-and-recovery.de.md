---
title-en: Attendees Signup and Recovery
title-de: Teilnehmende Signup und Recovery
---

# Teilnehmende, Signup und Recovery

## Routen für Teilnehmendenverwaltung

- Teilnehmende anlegen: `POST /committee/{slug}/meeting/{meeting_id}/attendee/create`
- Selbstanmeldung der Sitzungsleitung: `POST /committee/{slug}/meeting/{meeting_id}/attendee/self-signup`
- Teilnehmende entfernen: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/delete`
- Chair-Flag setzen: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/chair`
- Quoted-Flag setzen: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/quoted`

## Recovery-Unterstützung

- Recovery-Seite und QR: `GET /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/recovery`

Recovery-Link und QR nutzen, wenn Zugangscodes verloren wurden und kein Duplikat entstehen soll.
