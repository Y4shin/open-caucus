---
title-en: Committee Memberships and OAuth Rules
title-de: Gremiumsmitgliedschaften und OAuth-Regeln
---

# Gremiumsmitgliedschaften und OAuth-Regeln

Auf dieser Seite verwaltest du Single-Sign-On-Gruppenregeln und prüfst OAuth-verwaltete Mitgliedschaften.

> **Hinweis**: Die alltägliche Mitgliederverwaltung (Mitglieder hinzuf��gen, Rollen bearbeiten, Mitglieder entfernen) wurde auf die **Gremiumsseite der Sitzungsleitung** verschoben. Siehe [Gremiumsseite und Meeting-Lebenszyklus](/docs/03-chairperson/01-committee-dashboard-and-meeting-lifecycle) für Details. Die Admin-Seite behält die OAuth-Gruppenregel-Konfiguration.

## Für wen diese Seite gedacht ist

Admins, die Single-Sign-On-Gruppenzugriffsregeln für Gremien pflegen.

## Bevor du loslegst

1. Melde dich als Admin an.
2. Öffne das **Admin Dashboard** und wähle ein Gremium.
3. Stelle sicher, dass **Login with OAuth** in deiner Konfiguration aktiviert ist.

## Schritt für Schritt

1. Gremiums-OAuth-Regeln öffnen:
   - Gehe zum **Admin Dashboard**.
   - Klicke in der Gremiumszeile, um die Gremiums-Admin-Seite zu öffnen.
2. Login-Gruppenregel hinzufügen:
   - Gib unter **OAuth Group Access Rules** den **OAuth Group**-Namen ein.
   - Wenn `OAUTH_GROUP_PREFIX` konfiguriert ist (z.B. `committee-`), gleicht die Regel Gruppen mit diesem Präfix ab. Zum Beispiel passt mit dem Präfix `committee-` die OIDC-Gruppe `committee-finance` zur Regel `finance`.
   - Wähle die Rolle (`Member` oder `Chairperson`).
   - Klicke auf **Add Rule**.
3. Login-Gruppenregel entfernen:
   - Klicke in der Regeltabelle auf **Remove** für die nicht mehr benötigte Regel.
   - Bestätige die Abfrage.

## OIDC-Profilsynchronisation

Wenn sich ein Benutzer über OAuth/OIDC anmeldet:
- Die App ruft den **Userinfo-Endpunkt** des Identitätsanbieters ab und extrahiert den **Anzeigenamen** und die **E-Mail-Adresse** des Benutzers.
- Diese werden im Konto gespeichert und bei jedem Login aktualisiert, um Profile mit dem Identitätsanbieter synchron zu halten.
- Gremienmitgliedschaften werden basierend auf den OIDC-Gruppen des Benutzers und den hier konfigurierten Gruppenregeln synchronisiert.

## Was du sehen solltest

- Hinzugefügte Gruppenregeln erscheinen in der Regeltabelle und wirken auf zukünftige Anmeldungen.
- Einige Mitgliedschaften zeigen **Role managed by OAuth** — diese werden automatisch verwaltet.
- Benutzer, die sich über OIDC anmelden, erhalten ihren Anzeigenamen und ihre E-Mail-Adresse vom Anbieter aktualisiert.

## Falls etwas schiefgeht

- OAuth-Regelbereich fehlt:
  **Login with OAuth** ist in dieser Konfiguration nicht aktiviert.
- Gruppenregel erstellen/löschen schlägt fehl:
  Überprüfe Gruppennamen und Rollenwerte und versuche es erneut.
- Benutzer-Anzeigename zeigt die ID statt des echten Namens:
  Der Identitätsanbieter gibt möglicherweise Claims nicht korrekt zurück. Überprüfe, ob der OIDC-Provider `name` oder `preferred_username` im ID-Token oder Userinfo-Endpunkt zurückgibt.
- Gruppenprefix passt nicht:
  Überprüfe `OAUTH_GROUP_PREFIX` in der Serverkonfiguration — es muss zum Präfix der Gruppennamen deines Identitätsanbieters passen.

## Wie es weitergeht

Zurück zu [Konten und Gremienverwaltung](/docs/02-admin/02-accounts-and-committee-management) für Konto- und Gremiumseinrichtung, oder siehe [Gremiumsseite](/docs/03-chairperson/01-committee-dashboard-and-meeting-lifecycle) für die Mitgliederverwaltung.
