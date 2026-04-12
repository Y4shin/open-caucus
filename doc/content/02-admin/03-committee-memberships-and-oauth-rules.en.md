---
title-en: Committee Memberships and Sign-On Group Rules
title-de: Gremiumsmitgliedschaften und OAuth-Regeln
---

# Committee Memberships and Sign-On Group Rules

Use this page to manage single sign-on group rules and review OAuth-managed memberships.

> **Note**: Day-to-day member management (adding members, editing roles, removing members) has moved to the **chairperson committee page**. See [Committee Dashboard and Meeting Lifecycle](/docs/03-chairperson/01-committee-dashboard-and-meeting-lifecycle) for details. The admin page retains OAuth group rule configuration.

## Who this is for

Admins who maintain single sign-on group access rules for committees.

## Before you start

1. Sign in as admin.
2. Open **Admin Dashboard** and choose a committee.
3. Confirm **Login with OAuth** is enabled in your setup.

## Step-by-step

1. Open committee OAuth rules:
   - Go to **Admin Dashboard**.
   - In the committee row, click to open the committee admin page.
2. Add a login-group access rule:
   - In **OAuth Group Access Rules**, enter **OAuth Group** name.
   - If `OAUTH_GROUP_PREFIX` is configured (e.g. `committee-`), the rule matches groups with that prefix stripped. For example, with prefix `committee-`, the OIDC group `committee-finance` matches the rule `finance`.
   - Select role (`Member` or `Chairperson`).
   - Click **Add Rule**.
3. Remove a login-group rule:
   - In the rules table, click **Remove** for the rule you no longer need.
   - Confirm the prompt.

## OIDC profile sync

When a user logs in via OAuth/OIDC:
- The app fetches the identity provider's **userinfo endpoint** and extracts the user's **display name** and **email address**.
- These are stored on the account and updated on every login, keeping profiles in sync with the identity provider.
- Committee memberships are synchronized based on the user's OIDC groups and the group rules configured here.

## What you should see

- Added group rules appear in the rules table and affect future sign-ins.
- Some memberships show **Role managed by OAuth** — these are automatically maintained.
- Users who log in via OIDC will have their display name and email updated from the provider.

## If something goes wrong

- OAuth rules section is missing:
  **Login with OAuth** is not enabled in this setup.
- Group rule create/delete fails:
  Check group name and role values, then retry.
- User display name shows as their ID instead of their real name:
  The identity provider may not be returning claims correctly. Check that the OIDC provider returns `name` or `preferred_username` in either the ID token or the userinfo endpoint.
- Group prefix not matching:
  Verify `OAUTH_GROUP_PREFIX` in the server configuration matches the prefix used by your identity provider's group names.

## What happens next

Return to [Accounts and Committee Management](/docs/02-admin/02-accounts-and-committee-management) for account and committee setup, or see [Committee Dashboard](/docs/03-chairperson/01-committee-dashboard-and-meeting-lifecycle) for member management.
