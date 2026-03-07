---
title-en: Accounts and Committee Management
title-de: Accounts und Gremiumsverwaltung
---

# Accounts and Committee Management

Use this page to create accounts, create committees, and assign accounts to committees.

## Who this is for

Admins who prepare access and structure before committee work starts.

## Before you start

1. Sign in as admin and open the admin dashboard.
2. Decide which operation you need first:
   - Create user accounts
   - Create a committee
   - Assign accounts to a committee
3. Prepare required values:
   - Account: username, full name, and password (if password login is enabled)
   - Committee: committee name and short URL name

## Step-by-step

1. Create accounts:
   - Click **Manage Accounts**.
   - In **Add New Account**, fill **Username**, **Full Name**, and **Password** (when shown).
   - Click **Create Account**.
2. Create a committee:
   - Return to **Admin Dashboard**.
   - In **Add New Committee**, fill **Committee Name** and **Slug (URL-friendly identifier)**.
   - Click **Create Committee**.
3. Assign an account to a committee:
   - In **Existing Committees**, click **Assign Accounts** for the target committee.
   - In **Assign Account**, select **Account** and **Role**.
   - Optionally enable **FLINTA***.
   - Click **Assign Account**.
4. Adjust or remove existing assignments:
   - In **Assigned Accounts**, change role/FLINTA* values and click **Save**.
   - Click **Remove** for assignments you want to delete, then confirm.
5. Use **Delete** on a committee row only when you intentionally want to remove that committee.

## What you should see

- New accounts appear in **Existing Accounts**.
- New committees appear in **Existing Committees**.
- Newly assigned accounts appear in **Assigned Accounts** for the selected committee.
- Most create/update/remove actions refresh only the affected list area, not the whole page.

## If something goes wrong

- Create account fails:
  Confirm all required fields are filled and the username is not already used.
- Create committee fails:
  Confirm the short URL name is unique and uses only lowercase letters, numbers, and hyphens.
- No options are shown in the **Assign Account** selector:
  All accounts may already be assigned to that committee.
- You are unsure whether to delete a committee:
  Review committee usage first and delete only when you intend permanent removal.

## What happens next

Continue with [Committee Memberships and Sign-On Group Rules](/docs/02-admin/03-committee-memberships-and-oauth-rules) for role updates and login-group rules.
