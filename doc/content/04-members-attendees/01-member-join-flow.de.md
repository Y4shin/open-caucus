---
title-en: Member Join Flow
title-de: Mitglieds-Beitrittsablauf
---

# Mitglieds-Beitrittsablauf

![Mitglied von Join zu Live](../../assets/captures/app-member-join-to-live.en.light.desktop.gif)

## Ablauf

1. Mitglied meldet sich über `/` an.
2. Mitglied öffnet `/committee/{slug}`.
3. Aktiven-Meeting-Button anklicken.
4. Join-Submit sendet an `/committee/{slug}/meeting/{meeting_id}/join`.
5. Erfolgreicher Beitritt leitet zu `/committee/{slug}/meeting/{meeting_id}` weiter.

## Nicht erlaubtes Verhalten

- Inaktive Meetings sind nicht über den Aktiv-Shortcut beitretbar.
- Ohne Gremiumsmitgliedschaft greift `committee_access`.
