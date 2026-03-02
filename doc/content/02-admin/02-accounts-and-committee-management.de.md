---
title-en: Accounts and Committee Management
title-de: Accounts und Gremiumsverwaltung
---

# Accounts und Gremiumsverwaltung

## Gremiumsoperationen

- Gremium anlegen: `POST /admin/committee/create`
- Gremium löschen: `POST /admin/committee/{slug}/delete`
- Gremiumsseite: `GET /admin/committee/{slug}`

## Account-Operationen

- Account anlegen: `POST /admin/account/create`
- Paginierte Accountliste: `GET /admin/accounts`
- Account zu Gremium zuweisen: `POST /admin/committee/{slug}/account/assign`

## Validierung

- Slugs müssen eindeutig sein.
- Doppelte Usernames werden abgewiesen.
- HTMX aktualisiert nur betroffene Listenbereiche.
