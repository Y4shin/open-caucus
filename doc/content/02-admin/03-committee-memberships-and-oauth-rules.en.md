---
title-en: Committee Memberships and OAuth Rules
title-de: Gremiumsmitgliedschaften und OAuth-Regeln
---

# Committee Memberships and OAuth Rules

## Membership management

- Update membership role/quoted: `POST /admin/committee/{slug}/membership/{user_id}/update`
- Delete membership: `POST /admin/committee/{slug}/membership/{user_id}/delete`

## OAuth rule management

- Create rule: `POST /admin/committee/{slug}/oauth-group-rule/create`
- Delete rule: `POST /admin/committee/{slug}/oauth-group-rule/{rule_id}/delete`

## Operational notes

- OAuth-managed memberships can be protected from direct manual deletion.
- Group rule changes affect future OAuth sync and login provisioning behavior.
