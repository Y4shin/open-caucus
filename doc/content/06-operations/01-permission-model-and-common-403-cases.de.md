---
title-en: Permission Model and Common 403 Cases
title-de: Berechtigungsmodell und typische 403-Fälle
---

# Berechtigungsmodell und typische 403-Fälle

## Middleware-Gates

- `auth`, `session`, `admin_required`
- `committee_access`, `meeting_access`, `moderate_access`

## Häufige 403-Ursachen

- Zugriff auf `/admin` ohne Adminrolle.
- Zugriff auf `/committee/{slug}` ohne Gremiumszugehörigkeit.
- Zugriff auf Moderation ohne Chair-Berechtigung.
- Fehlende Verknüpfung als Meeting-Teilnehmende für Live-Aktionen.

## Betriebscheck

Bei Rechteproblemen nacheinander Rolle, Gremiumsmitgliedschaft, Attendee-Verknüpfung und Session-Identität prüfen.
