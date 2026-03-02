---
title-en: Login and Session Model
title-de: Login- und Sitzungsmodell
---

# Login and Session Model

## Authentication paths

- Admin credentials submit to `/admin/login`.
- Committee member credentials submit to `/login` from `/`.
- Guest attendees authenticate with access code at `/committee/{slug}/meeting/{meeting_id}/attendee-login`.

## Session behavior

- Existing valid sessions skip redundant login pages.
- Role-bound session context drives visible actions in templates.
- Logout endpoints:
  - Admin logout: `/admin/logout`
  - User/attendee logout: `/logout`

## Security expectations

- Wrong credentials remain on the same form with an error message.
- Protected pages without session redirect to the corresponding login flow.
