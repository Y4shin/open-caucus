---
title-en: Speakers Moderator and Quotation
title-de: Redeliste Moderation und Quotierung
---

# Speakers Moderator and Quotation

## Speaker routes

- Add speaker: `POST /committee/{slug}/meeting/{meeting_id}/speaker/add`
- Remove: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/remove`
- Start: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/start`
- End: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/end`
- Withdraw: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/withdraw`
- Priority toggle: `POST /committee/{slug}/meeting/{meeting_id}/speaker/{speaker_id}/priority`

## Quotation and moderator settings

- Meeting quotation defaults: `POST /committee/{slug}/meeting/{meeting_id}/quotation`
- Meeting moderator override: `POST /committee/{slug}/meeting/{meeting_id}/moderator`
- Agenda quotation override: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/quotation`
- Agenda moderator override: `POST /committee/{slug}/meeting/{meeting_id}/agenda-point/{agenda_point_id}/moderator`
