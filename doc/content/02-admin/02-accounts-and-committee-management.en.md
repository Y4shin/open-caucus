---
title-en: Accounts and Committee Management
title-de: Accounts und Gremiumsverwaltung
---

# Accounts and Committee Management

## Committee operations

- Create committee: `POST /admin/committee/create`
- Delete committee: `POST /admin/committee/{slug}/delete`
- Committee page: `GET /admin/committee/{slug}`

## Account operations

- Create account: `POST /admin/account/create`
- Paginated account listing: `GET /admin/accounts`
- Assign account to committee: `POST /admin/committee/{slug}/account/assign`

## Validation expectations

- Slugs must be unique.
- Duplicate usernames are rejected.
- HTMX updates only affected list fragments; full-page reload is not expected.
