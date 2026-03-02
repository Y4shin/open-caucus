---
title-en: Join QR and Access Hand Off
title-de: Join QR und Zugangsübergabe
---

# Join-QR und Übergabe von Zugängen

## QR-gestützter Beitritt

- Join-QR-Seite: `GET /committee/{slug}/meeting/{meeting_id}/moderate/join-qr`

Der QR öffnet die Join-Seite mit vorausgefülltem Secret und reduziert Eingabefehler.

## Zugang an Gäste übergeben

- Gast-Signup: `POST /committee/{slug}/meeting/{meeting_id}/guest`
- Attendee-Loginseite: `GET /committee/{slug}/meeting/{meeting_id}/attendee-login`

Nach dem Signup erhalten Teilnehmende einen individuellen Zugangscode, der wie ein Login-Credential zu behandeln ist.
