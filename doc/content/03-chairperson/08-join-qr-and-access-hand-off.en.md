---
title-en: Join QR and Access Hand Off
title-de: Join QR und Zugangsübergabe
---

# Join QR and Access Hand-Off

## QR-based join support

- Join QR page: `GET /committee/{slug}/meeting/{meeting_id}/moderate/join-qr`

Use this QR when guests should open the join page with prefilled secret. This reduces typing errors at entry desks.

## Guest access hand-off

- Guest signup endpoint: `POST /committee/{slug}/meeting/{meeting_id}/guest`
- Attendee login page: `GET /committee/{slug}/meeting/{meeting_id}/attendee-login`

After signup, attendees receive a unique access code. Treat it as a session credential.
