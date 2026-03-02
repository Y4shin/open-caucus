---
title-en: Access and Roles
title-de: Zugriff und Rollen
---

# Zugriff und Rollen

## Rollenübersicht

- `admin`: globale Verwaltung, Accounts, Gremien, OAuth-Regeln.
- `chairperson`: Gremiumsverwaltung, Moderation, Tagesordnung und Abstimmungen.
- `member`: Teilnahme im Gremium und Mitglieds-Beitritt.
- `guest attendee`: Beitritt über Gastformular und Zugangscode.

## Einstiegsrouten

- Admin-Login: `/admin/login`
- Benutzer-Login: `/`
- Gremiumsseite: `/committee/{slug}`
- Join-Seite: `/committee/{slug}/meeting/{meeting_id}/join`
- Teilnehmenden-Login: `/committee/{slug}/meeting/{meeting_id}/attendee-login`

## Rechteprüfung

Nicht erlaubte Zugriffe führen je nach Middleware zu `403` oder zur Login-Weiterleitung.
