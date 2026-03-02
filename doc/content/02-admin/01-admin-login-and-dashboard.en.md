---
title-en: Admin Login and Dashboard
title-de: Admin-Login und Dashboard
---

# Admin Login and Dashboard

## Primary routes

- Login form: `/admin/login`
- Dashboard: `/admin`
- Accounts index: `/admin/accounts`

## Typical workflow

1. Log in with an admin account.
2. Review committees and accounts from the dashboard cards and lists.
3. Navigate into committee-specific management pages.

## Guardrails

- Non-admin sessions are rejected by `admin_required` middleware.
- Password auth disabled state blocks password-based admin submit.
