---
title-en: Access and Roles
title-de: Zugriff und Rollen
---

# Access and Roles

## Role overview

- `admin`: global administration, accounts, committees, OAuth rules.
- `chairperson`: committee management, meeting moderation, agenda and vote control.
- `member`: committee participation and member join flow.
- `guest attendee`: joins through meeting guest signup and access code login.

## Entry routes

- Admin login: `/admin/login`
- User login: `/`
- Committee dashboard: `/committee/{slug}`
- Join page: `/committee/{slug}/meeting/{meeting_id}/join`
- Attendee login: `/committee/{slug}/meeting/{meeting_id}/attendee-login`

## Access enforcement

Unauthorized route access returns `403` or redirects to login, depending on route middleware and session state.
