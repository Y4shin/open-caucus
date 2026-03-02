---
title-en: Committee Memberships and OAuth Rules
title-de: Gremiumsmitgliedschaften und OAuth-Regeln
---

# Gremiumsmitgliedschaften und OAuth-Regeln

## Mitgliedschaften verwalten

- Rolle/Quoted aktualisieren: `POST /admin/committee/{slug}/membership/{user_id}/update`
- Mitgliedschaft löschen: `POST /admin/committee/{slug}/membership/{user_id}/delete`

## OAuth-Regeln verwalten

- Regel anlegen: `POST /admin/committee/{slug}/oauth-group-rule/create`
- Regel löschen: `POST /admin/committee/{slug}/oauth-group-rule/{rule_id}/delete`

## Betriebshinweise

- OAuth-verwaltete Mitgliedschaften können gegen manuelles Löschen geschützt sein.
- Änderungen an Gruppenregeln wirken auf zukünftige OAuth-Synchronisationen.
