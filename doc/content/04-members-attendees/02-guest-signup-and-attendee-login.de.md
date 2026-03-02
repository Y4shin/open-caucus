---
title-en: Guest Signup and Attendee Login
title-de: Gast-Signup und Attendee-Login
---

# Gast-Signup und Attendee-Login

![Gast-Signup Formular](../../assets/captures/app-guest-signup-form.en.light.desktop.png)

![Attendee-Login](../../assets/captures/app-attendee-login.en.light.desktop.png)

## Gastablauf

- Join-Seite öffnen: `GET /committee/{slug}/meeting/{meeting_id}/join`
- Gastformular senden: `POST /committee/{slug}/meeting/{meeting_id}/guest`
- Zugangscode eingeben: `POST /committee/{slug}/meeting/{meeting_id}/attendee-login`

## Recovery und Sicherheit

- Zugangscode nur für das zugehörige Meeting verwenden.
- Bei Verlust Recovery-Link/QR über die Sitzungsleitung nutzen.
