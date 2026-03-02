---
title-en: Admin Login and Dashboard
title-de: Admin-Login und Dashboard
---

# Admin-Login und Dashboard

## Wichtige Routen

- Loginformular: `/admin/login`
- Dashboard: `/admin`
- Account-Übersicht: `/admin/accounts`

## Typischer Ablauf

1. Mit einem Admin-Account anmelden.
2. Gremien und Accounts im Dashboard prüfen.
3. In gremiumsspezifische Verwaltungsseiten wechseln.

## Schutzmechanismen

- Nicht-Admin-Sessions werden durch `admin_required` abgewiesen.
- Wenn Passwortauth deaktiviert ist, sind Passwort-Submits blockiert.
