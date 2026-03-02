---
title-en: Live Page Self Service and Voting
title-de: Live-Seite Self-Service und Abstimmung
---

# Live-Seite, Self-Service und Abstimmung

![Live-Ansicht mit Redeliste](../../assets/captures/app-live-view-with-speakers.en.light.desktop.png)

## Self-Service in der Redeliste

- Sich selbst melden: `POST /committee/{slug}/meeting/{meeting_id}/speaker/self-add`
- Eigene Rede beenden: `POST /committee/{slug}/meeting/{meeting_id}/speaker/self-yield`

## Live-Abstimmung

- Panel aktualisieren: `GET /committee/{slug}/meeting/{meeting_id}/votes/live/partial`
- Offene Stimme senden: `POST /committee/{slug}/meeting/{meeting_id}/votes/{vote_id}/submit/open`
- Geheime Stimme senden: `POST /committee/{slug}/meeting/{meeting_id}/votes/{vote_id}/submit/secret`

## Grenzen

- Geschlossene oder archivierte Abstimmungen nehmen keine Stimmen an.
- Min-/Max-Auswahlregeln der Abstimmung müssen eingehalten werden.
