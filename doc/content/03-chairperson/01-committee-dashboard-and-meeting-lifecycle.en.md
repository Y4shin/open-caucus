---
title-en: Committee Dashboard and Meeting Lifecycle
title-de: Gremiumsseite und Meeting-Lebenszyklus
---

# Committee Dashboard and Meeting Lifecycle

![Chair committee dashboard](../../assets/captures/app-committee-dashboard-chair.en.light.desktop.png)

Use this page to create meetings, choose the current one, and control when participants can sign up.

## Who this is for

Chairpersons who run committee meetings and need to control meeting status.

## Before you start

1. Open your committee dashboard as a chairperson.
2. Decide whether you are preparing a new meeting or updating an existing one.
3. Prepare a clear meeting name and optional short description.

## Step-by-step

1. Create a meeting:
   - Click the **Create Meeting** button to open the meeting wizard.
   - The wizard guides you through up to four steps:
     1. **Basics** — enter **Name** and optional **Description**. Optionally set a **Start** and **End** date/time using the date range picker. Enable **Open for signup** if participants should be able to join immediately.
     2. **Agenda** — optionally enter the agenda. Type or paste items one per line; use numbering (1.1, 1.2) or indentation for sub-items. Click the chips on the right to classify each line as heading, subheading, or ignore. Switch between Plaintext and Markdown format with the toggle.
     3. **Invites** — select which committee members should receive an invite email. All members are selected by default. Members without an email address are shown but greyed out. Choose a **Language** (EN/DE) and **Timezone** for the email, and optionally add a **Custom message**. If email sending is not configured, this step is skipped.
     4. **Review** — confirm all details and click **Create Meeting**. If invites are enabled, emails with ICS calendar attachments are sent automatically after creation.
2. Edit a meeting:
   - Click the **Edit** button on a meeting row to change its name, description, or date/time after creation.
3. Manage committee members:
   - Below the meetings list, the **Members** panel lets you manage who belongs to this committee.
   - **Add by email**: Enter an email address, full name, and role. The member does not need a sitewide account — they will receive a personalized invite link.
   - **Assign account**: Select an existing account from the dropdown and assign a role.
   - Edit roles, quotation status, or remove members from the member table.
4. Send invite emails:
   - From the meeting moderation page, use **Send Invites** to email meeting invitations to selected members.
   - Choose language, timezone, and add an optional custom message.
   - Emails include an ICS calendar attachment when the meeting has a date/time set.
   - Email-only members receive a personalized link with an invite secret. Account-based members receive a direct meeting link.
5. Set the current meeting:
   - In the meetings list, switch on **Active** for the meeting you are running now.
   - Confirm only one meeting is marked **Active**.
6. Control participant entry:
   - Use the **Signup Open** switch for the active meeting.
   - Keep it enabled while people are joining, then disable it when entry should stop.
7. Open meeting tools:
   - Use **Manage** on the meeting row to open moderation controls.
   - Use **View** to check the live participant-facing page.
8. Close out old or test meetings:
   - Click **Delete** on meetings you no longer need and confirm.

## What you should see

- New meetings appear in the meeting list immediately.
- Active and signup status changes directly in the list when switches are changed.
- **Manage** opens chairperson controls and **View** opens the live meeting page.

## If something goes wrong

- Participants cannot join:
  Check that the correct meeting is **Active** and **Signup Open** is enabled.
- People join the wrong room:
  Verify only the intended meeting is marked **Active**.
- Meeting create action fails:
  Recheck required fields (meeting name) and submit again.
- You are unsure whether to delete:
  Keep past meetings until results/records are no longer needed.

## What happens next

Continue with [Moderation Page Overview](/docs/03-chairperson/02-moderate-workspace-overview) to run the active meeting.
