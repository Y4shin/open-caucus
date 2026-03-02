---
title-en: Capture Guide
title-de: Capture-Leitfaden
---

# Capture-Leitfaden

## Script-Registry

Unterstützt werden ausschließlich `app.*` Scripts:

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

## Befehle

```bash
go run . docs-capture list --script '*'
go run . docs-capture run --script 'app.screenshot-*' --theme light --language en --device desktop
go run . docs-capture run --script 'app.gif-*' --theme light --language en --device desktop
```

## Ausgabe-Konventionen

- Captures liegen in `doc/assets/captures/`.
- Dateinamen enthalten Suffixe für Sprache, Theme und Gerät.
- Markdown referenziert Assets relativ unter `assets/captures/`.
