---
title-en: Capture Guide
title-de: Capture-Leitfaden
---

# Capture Guide

## Script registry

Only `app.*` scripts are supported:

- `app.screenshot-home-committees`
- `app.screenshot-committee-dashboard-chair`
- `app.screenshot-committee-dashboard-member-active`
- `app.screenshot-moderate-overview`
- `app.screenshot-agenda-tools-attachments`
- `app.screenshot-live-view-with-speakers`
- `app.screenshot-join-page-member`
- `app.screenshot-guest-signup-form`
- `app.screenshot-attendee-login`
- `app.screenshot-receipts-vault`
- `app.gif-member-join-to-live`
- `app.gif-speaker-lifecycle-moderate-to-live`
- `app.gif-vote-lifecycle-open-and-secret`

## Commands

```bash
go run . docs-capture list --script '*'
go run . docs-capture run --script 'app.screenshot-*' --theme light --language en --device desktop
go run . docs-capture run --script 'app.gif-*' --theme light --language en --device desktop
```

## Output conventions

- Captures are written to `doc/assets/captures/`.
- Filenames include language, theme, and device suffixes.
- Markdown should reference capture assets under `assets/captures/` with relative paths.
