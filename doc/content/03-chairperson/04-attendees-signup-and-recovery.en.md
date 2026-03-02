---
title-en: Attendees Signup and Recovery
title-de: Teilnehmende Signup und Recovery
---

# Attendees Signup and Recovery

## Attendee management routes

- Add attendee: `POST /committee/{slug}/meeting/{meeting_id}/attendee/create`
- Chairperson self-signup: `POST /committee/{slug}/meeting/{meeting_id}/attendee/self-signup`
- Remove attendee: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/delete`
- Toggle chair: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/chair`
- Toggle quoted: `POST /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/quoted`

## Recovery support

- Recovery page and QR: `GET /committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/recovery`

Use recovery link/QR when attendees lost their access code but must re-enter without duplicate signup.
