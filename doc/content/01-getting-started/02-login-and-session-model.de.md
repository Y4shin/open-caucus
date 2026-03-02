---
title-en: Login and Session Model
title-de: Login- und Sitzungsmodell
---

# Login- und Sitzungsmodell

## Authentifizierungspfade

- Admin-Credentials werden über `/admin/login` gesendet.
- Mitglieds-Credentials werden über `/login` von `/` gesendet.
- Gäste melden sich mit Zugangscode über `/committee/{slug}/meeting/{meeting_id}/attendee-login` an.

## Sitzungsverhalten

- Bei gültiger Session werden Loginseiten übersprungen.
- Der rollenbezogene Session-Kontext steuert sichtbare Aktionen.
- Logout-Endpunkte:
  - Admin-Logout: `/admin/logout`
  - User-/Attendee-Logout: `/logout`

## Sicherheitserwartung

- Falsche Zugangsdaten bleiben auf dem Formular und zeigen einen Fehler.
- Geschützte Seiten ohne Session leiten zum passenden Loginfluss weiter.
