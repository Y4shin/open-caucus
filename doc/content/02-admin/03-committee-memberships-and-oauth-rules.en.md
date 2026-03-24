---
title-en: Committee Memberships and Sign-On Group Rules
title-de: Gremiumsmitgliedschaften und OAuth-Regeln
---

# Committee Memberships and Sign-On Group Rules

Use this page to manage committee memberships and, if enabled, set single sign-on group rules.

## Who this is for

Admins who manage roles inside committees and admins who maintain single sign-on group access.

## Before you start

1. Sign in as admin.
2. Open **Admin Dashboard** and choose a committee via **Assign Accounts**.
3. If you plan to manage group rules, confirm **Login with OAuth** is enabled in your setup.

## Step-by-step

1. Open committee membership management:
   - Go to **Admin Dashboard**.
   - In the committee row, click **Assign Accounts**.
2. Update an existing membership:
   - In **Assigned Accounts**, choose role (`Member` or `Chairperson`).
   - Update **FLINTA*** if needed.
   - Click **Save**.
3. Remove a membership assignment:
   - In the same row, click **Remove**.
   - Confirm the prompt.
4. Add a login-group access rule (when OAuth login is enabled):
   - In **OAuth Group Access Rules**, enter **OAuth Group**.
   - Select role (`Member` or `Chairperson`).
   - Click **Add Rule**.
5. Remove a login-group rule:
   - In the rules table, click **Remove** for the rule you no longer need.
   - Confirm the prompt.

## What you should see

- Membership changes are reflected in the **Assigned Accounts** table.
- Some memberships can show **Role managed by OAuth** and may block manual role edits.
- Added group rules appear in the rules table and affect future sign-ins.

## If something goes wrong

- Save is accepted but role does not change:
  The membership may be managed automatically; check for the **Role managed by OAuth** hint.
- Remove membership fails:
  Automatic management or current access rules may block direct removal.
- OAuth rules section is missing:
  **Login with OAuth** is not enabled in this setup.
- Group rule create/delete fails:
  Check group name and role values, then retry.

## What happens next

Return to [Accounts and Committee Management](/docs/02-admin/02-accounts-and-committee-management) for account and committee setup, or continue to role guides.
