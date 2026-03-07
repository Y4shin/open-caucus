---
title-en: Permission Model and Common 403 Cases
title-de: Berechtigungsmodell und typische 403-Fälle
---

# Permission Model and Common 403 Cases

Use this guide when someone cannot open a page because of missing permissions.

## Who this is for

Admins, chairpersons, and support operators handling access issues.

## Before you start

1. Ask which page the user tried to open.
2. Confirm which account/session they used:
   - admin account
   - committee user
   - attendee session
3. Confirm committee and meeting context.

## Quick permission map

1. **Admin pages** require an admin session.
2. **Committee pages** require membership in that committee.
3. **Manage** and **Moderate** pages require chairperson-level access, or matching attendee-level authority.
4. **Meeting pages for regular members** usually require the meeting to be currently active.

## Common 403/Access-Denied Cases

1. Non-admin user opens admin page.
2. User opens another committee's page.
3. Member (non-chair) tries to open manage/moderate pages.
4. Attendee session belongs to a different meeting than the requested page.
5. User tries to open a non-active meeting page as regular member.

## Step-by-step diagnosis

1. Confirm whether the issue is a redirect to login or a true access-denied response.
2. Check role first:
   - admin
   - chairperson
   - member
   - attendee
3. Check committee match.
4. Check meeting match (for attendee sessions).
5. Re-test with the correct account/session.

## If something goes wrong

- User insists they have access:
  Verify they are signed in with the expected account (not an old browser session).
- Chairperson can access one meeting but not another:
  Check committee and meeting assignment/matching.
- Attendee cannot open moderation page:
  Confirm they are actually marked as chair (or designated moderator) for that meeting.
- Access issue appears only on one device:
  Ask the user to log out and log in again on that device.
