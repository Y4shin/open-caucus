---
title-en: Guest Signup and Attendee Login
title-de: Gast-Signup und Attendee-Login
---

# Guest Signup and Attendee Login

![Guest signup form](../../assets/captures/app-guest-signup-form.en.light.desktop.png)

![Attendee login](../../assets/captures/app-attendee-login.en.light.desktop.png)

## Guest route sequence

- Open join page: `GET /committee/{slug}/meeting/{meeting_id}/join`
- Submit guest form: `POST /committee/{slug}/meeting/{meeting_id}/guest`
- Enter access code: `POST /committee/{slug}/meeting/{meeting_id}/attendee-login`

## Recovery and safety

- Use provided access code only for the matching meeting.
- If the code is lost, chairperson can provide a recovery link/QR from moderate attendee tools.
