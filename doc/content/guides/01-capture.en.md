---
title-en: Capture Guide
title-de: Capture-Leitfaden
---

# Capture Guide

Use this guide to generate screenshots and GIFs used in documentation.

## Who this is for

Documentation maintainers updating visual assets.

## Before you start

1. Start from the project root.
2. Ensure the app can run locally.
3. Decide what you need:
   - a screenshot
   - a GIF
4. Use the same theme/language/device settings as the target docs.

## Step-by-step

1. List available capture scripts:

```bash
go run . docs-capture list --script '*'
```

2. Generate screenshots:

```bash
go run . docs-capture run --script 'app.screenshot-*' --theme light --language en --device desktop
```

3. Generate GIFs:

```bash
go run . docs-capture run --script 'app.gif-*' --theme light --language en --device desktop
```

## Supported script set

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

## What you should see

- Captures are written to `doc/assets/captures/`.
- Filenames include language, theme, and device suffixes.
- Markdown should reference capture assets under `assets/captures/` with relative paths.

## If something goes wrong

- Script not found:
  Run the list command again and verify the script name.
- Wrong language/theme/device in output:
  Re-run with explicit flags and replace the incorrect files.
- Capture does not match current UI:
  Update the script flow first, then capture again.
