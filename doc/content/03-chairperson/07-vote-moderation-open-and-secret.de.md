---
title-en: Vote Moderation Open and Secret
title-de: Abstimmungsmoderation offen und geheim
---

# Abstimmungsmoderation offen und geheim

![Abstimmungsablauf GIF](../../assets/captures/app-vote-lifecycle-open-and-secret.en.light.desktop.gif)

## Moderationsrouten

- Draft erstellen: `POST /committee/{slug}/meeting/{meeting_id}/votes/create`
- Draft aktualisieren: `POST .../votes/{vote_id}/update-draft`
- Abstimmung öffnen: `POST .../votes/{vote_id}/open`
- Abstimmung schließen: `POST .../votes/{vote_id}/close`
- Abstimmung archivieren: `POST .../votes/{vote_id}/archive`

## Auszählungsrouten

- Stimmabgabe registrieren (geheim): `POST .../votes/{vote_id}/cast/register`
- Geheime Stimmzettel zählen: `POST .../votes/{vote_id}/ballot/secret`
- Offene Stimmzettel zählen: `POST .../votes/{vote_id}/ballot/open`

## Teilnehmendenrouten

- Offene Stimme senden: `POST .../votes/{vote_id}/submit/open`
- Geheime Stimme senden: `POST .../votes/{vote_id}/submit/secret`
