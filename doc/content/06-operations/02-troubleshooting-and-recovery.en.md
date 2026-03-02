---
title-en: Troubleshooting and Recovery
title-de: Troubleshooting und Wiederherstellung
---

# Troubleshooting and Recovery

## Typical incidents

- attendees lost access codes
- wrong active meeting selected
- stale browser state after role/session switches
- incomplete vote flow (left in counting state)

## Recovery actions

- attendee recovery link/QR: `/committee/{slug}/meeting/{meeting_id}/attendee/{attendee_id}/recovery`
- reset active meeting from committee dashboard toggle
- refresh HTMX panels (`Refresh` controls) before retrying critical actions
- archive finalized votes to close operator loops

## Documentation integrity checks

- ensure every docs directory has `index` in EN+DE
- ensure every page has `title-en` and `title-de`
- run docs tests before deployment
