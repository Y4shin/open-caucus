---
title-en: Moderate Workspace Overview
title-de: Moderationsoberfläche im Überblick
---

# Moderate Workspace Overview

![Moderate overview](../../assets/captures/app-moderate-overview.en.light.desktop.png)

## Main route

- Moderate page: `/committee/{slug}/meeting/{meeting_id}/moderate`

## Workspace regions

- left controls: agenda, tools, attendees, settings tabs
- center/right panels: speakers queue, attendee quick add, vote panel
- SSE updates: `/committee/{slug}/meeting/{meeting_id}/moderate/stream`

## Operator expectation

Use moderate page for all real-time actions; it is optimized for HTMX partial updates and SSE refresh.
