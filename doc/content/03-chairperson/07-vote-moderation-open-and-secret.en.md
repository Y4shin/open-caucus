---
title-en: Vote Moderation Open and Secret
title-de: Abstimmungsmoderation offen und geheim
---

# Vote Moderation Open and Secret

![Vote lifecycle gif](../../assets/captures/app-vote-lifecycle-open-and-secret.en.light.desktop.gif)

## Moderator routes

- Create vote draft: `POST /committee/{slug}/meeting/{meeting_id}/votes/create`
- Update draft: `POST .../votes/{vote_id}/update-draft`
- Open vote: `POST .../votes/{vote_id}/open`
- Close vote: `POST .../votes/{vote_id}/close`
- Archive vote: `POST .../votes/{vote_id}/archive`

## Counting routes

- Register cast (secret flow): `POST .../votes/{vote_id}/cast/register`
- Count secret ballot: `POST .../votes/{vote_id}/ballot/secret`
- Count open ballot: `POST .../votes/{vote_id}/ballot/open`

## Attendee submission routes

- Open ballot submit: `POST .../votes/{vote_id}/submit/open`
- Secret ballot submit: `POST .../votes/{vote_id}/submit/secret`
