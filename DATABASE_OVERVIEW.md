# Database Structure Overview

This conference management tool supports the following workflow:

## Core Hierarchy

**Committees** → **Meetings** → **Agenda Points** → **Discussions & Motions**

## Key Entities

### Users & Participants
- **Users**: Committee members with accounts (chairperson or member roles)
  - Can be marked as on the **quoted speakers list** (priority speakers)
- **Attendees**: Meeting participants (may be registered users or guests)
  - Can be designated as **meeting chairs** (multiple per meeting)
  - One can be assigned as **protocol writer**
  - Can be marked as on the **quoted speakers list** (priority speakers)
  - Guests have unique secret, which can be used to recover access if device has to be switched or similar

### Meeting Structure
- **Meetings**: Belong to committees, have signup controls and unique access secrets
  - Secret is used for allowing guests to join (url with query param, can be used as QR code)
  - Tracks current agenda point being discussed
  - Optional protocol writer assignment
- **Agenda Points**: Hierarchical items (support nested sub-points)
  - Each has its own protocol section
  - Tracks current active speaker

### Discussion Management
- **Speakers List**: Queue of attendees wanting to speak on an agenda point
  - Types: regular speakers or "rules of procedure motion" (ROPM)
  - States: waiting, speaking, done, withdrawn
  - Records speech timing
- **Motions**: Proposals with attached documents
  - Optional voting results (for/against/abstained/eligible)

### File Management
- **Binary Blobs**: Stores uploaded files
- **Agenda Attachments**: Links files to specific agenda points

## Workflow Example

1. Committee creates a meeting with signup enabled
2. Users register as attendees (or guests are added)
3. Chairs and protocol writer are designated
4. Meeting progresses through agenda points
5. Attendees join speakers lists to contribute
6. Chairs manage speaker queue and advance agenda
7. Motions are proposed, voted on, and recorded
8. Protocol writer documents the proceedings
