---
title-en: Patch Notes
title-de: Versionshinweise
---

# Versionshinweise

## v1.4.5 — 2026-04-12

### Verbesserungen

- **Dokumentation**: Versionshinweise zur In-App-Dokumentation hinzugefuegt. Dokumentation fuer Sitzungsleitung, Admin und Quotierung aktualisiert, um v1.4.0-Aenderungen widerzuspiegeln (Mitgliederverwaltung, Einladungs-E-Mails, Quotierungsregeln, OIDC-Profilsynchronisation).
- **UI-Bezeichnung**: "Quotiert"-Label durch "FLINTA*" im Mitgliederverwaltungspanel ersetzt, um mit dem Rest der Anwendung uebereinzustimmen.
- **CLAUDE.md**: Regeln fuer Pflege der Versionshinweise bei jedem Release und Dokumentationsaktualisierung bei neuen Features hinzugefuegt.

---

## v1.4.4 — 2026-04-12

### Fehlerbehebungen

- **E-Mail-Header**: RFC 2822 `Date`-Header zu ausgehenden E-Mails hinzugefuegt, um Ablehnung durch Mail-Relays zu verhindern.

---

## v1.4.3 — 2026-04-12

### Fehlerbehebungen

- **E-Mail-Threading**: RFC 5322 `Message-ID`-Header zu ausgehenden Einladungs-E-Mails hinzugefuegt. E-Mails ohne diesen Header wurden von Mail-Relays abgelehnt (z.B. amavis `BAD-HEADER-0`).
- **E-Mail-Threading**: Gesendete E-Mails werden in neuer `sent_emails`-Tabelle erfasst. Der `References`-Header wird gesetzt, damit Einladungs-E-Mails desselben Gremiums im E-Mail-Client des Empfaengers als Thread gruppiert werden.

---

## v1.4.2 — 2026-04-12

### Verbesserungen

- **OIDC-Logging**: Umfassendes Debug-Logging fuer OIDC-Gruppenerkennung, Gremien-Synchronisation und E-Mail-Einladungsversand zur Fehleranalyse in Produktivumgebungen.

---

## v1.4.1 — 2026-04-12

### Fehlerbehebungen

- **OIDC-Profilsynchronisation**: Der OIDC-Userinfo-Endpunkt wird nun bei jedem Login abgefragt und Anzeigename sowie E-Mail-Adresse des Kontos aktualisiert. Zuvor wurde nur das ID-Token gelesen und Profile nur fuer neue Konten gesetzt, was zu veralteten oder fehlenden Anzeigenamen fuehrte.

---

## v1.4.0 — 2026-04-12

### Neue Funktionen

- **Mitgliederverwaltung fuer Vorsitzende**: Vorsitzende koennen Gremienmitglieder nun direkt auf der Gremien-Seite verwalten, ohne seitenweite Admin-Rechte zu benoetigen. Mitglieder per E-Mail hinzufuegen, bestehende Konten zuweisen, Rollen bearbeiten und Mitglieder entfernen.
- **E-Mail-basierte Mitglieder**: Gremienmitglieder benoetigen kein seitenweites Konto mehr. Vorsitzende koennen Mitglieder per E-Mail-Adresse hinzufuegen; diese erhalten einen personalisierten Einladungslink.
- **Sitzungs-Einladungs-E-Mails**: Einladungs-E-Mails an Gremienmitglieder mit ICS-Kalenderanhaengen, Tagesordnungsuebersicht, benutzerdefinierter Nachricht, Sprachauswahl (DE/EN) und Zeitzonen-Unterstuetzung.
- **Sitzungserstellungs-Assistent — Einladungsschritt**: Der Assistent enthaelt nun einen Einladungsschritt, in dem Vorsitzende auswaehlen, welche Mitglieder Einladungs-E-Mails erhalten. Ersetzt die alte textbasierte Teilnehmereingabe.
- **Datumsbereichsauswahl**: Sitzungen unterstuetzen nun optionale Start- und Enddatumszeit mit einem kalenderbasierten Datumsbereichswaehler.
- **Sitzungsbearbeitung**: Sitzungsdetails (Name, Beschreibung, Datum/Zeit) nach Erstellung bearbeiten.
- **Quotierungsregeln fuer Redeliste**: Ueberarbeitetes Quotierungssystem mit sortierbaren Drag-and-Drop-Regeln und einer animierten Schritt-fuer-Schritt-Visualisierung der Sortierlogik.
- **bits-ui Migration**: Interaktive UI-Komponenten (Select, Switch, Collapsible, Dialog, Tooltip, DatePicker, DateRangePicker) von DaisyUI zu bits-ui migriert fuer verbesserte Barrierefreiheit (ARIA-Rollen, Tastaturnavigation).
- **Tagesordnungsimport — Titelhervorhebung**: Extrahierte Titel in der Tagesordnungsimport-Vorschau werden nun im Bearbeitungstextfeld hervorgehoben.
- **OIDC-E-Mail-Extraktion**: Der OIDC-Login-Ablauf extrahiert und speichert nun die E-Mail-Adresse des Benutzers vom Identitaetsanbieter.
- **OIDC-Gruppenpraefix**: Konfigurierbares `OAUTH_GROUP_PREFIX` zum Filtern, welche OIDC-Gruppen fuer die Gremien-Synchronisation beruecksichtigt werden.
- **Betriebshandbuch**: Umfassende Deployment-Dokumentation fuer Systemadministratoren mit allen Umgebungsvariablen, OIDC-, E-Mail- und Mitgliederverwaltungskonfiguration.

### Fehlerbehebungen

- **Sitzungsgeheimnisse**: Automatische Generierung eines Sitzungsgeheimnisses bei der Erstellung. Bestehende Sitzungen ohne Geheimnis werden beim Start nachgefuellt.
- **Docscapture-Selektoren**: Alle Screenshot-Selektoren fuer die Dokumentation an die bits-ui-Migration angepasst (Collapsible, Dialog, etc.).

---

## v1.3.0 — 2026-04-10

### Neue Funktionen

- **OIDC-Gruppenpraefix-Konfiguration**: Konfigurierbares Praefix fuer OIDC-Gruppennamen in Gremien-Synchronisationsregeln.
- **OIDC-Debug-Logging**: Debug-Level-Logging fuer OIDC-Gruppenerkennung und Gremienmitgliedschafts-Synchronisation.

---

## v1.2.1 — 2026-04-10

### Neue Funktionen

- **Webhook-Header-Authentifizierung**: Unterstuetzung fuer benutzerdefinierte Header pro URL in `WEBHOOK_URLS` fuer authentifizierten Webhook-Versand.

### Fehlerbehebungen

- **Docker-Build-Kontext**: Die Verzeichnisse `doc/` und `tools/` werden nicht mehr vom Docker-Build-Kontext ausgeschlossen, wodurch die Dokumentationserfassung in CI wieder funktioniert.

---

## v1.2.0 — 2026-04-06

### Neue Funktionen

- **Ausgehende Webhooks**: Webhook-Dispatcher fuer Sitzungsereignisse (Sitzung erstellt, gestartet, beendet) und Gremien-/OIDC-Gruppen-Ereignisse. Konfiguration ueber die Umgebungsvariable `WEBHOOK_URLS`.
- **Webhook-Dokumentation**: `WEBHOOKS.md` mit Ereignisschemas und Konfigurationsanleitung hinzugefuegt.

### Fehlerbehebungen

- **OAuth-Logging**: Issuer, Benutzername und Gruppen werden nun in Login-Fehlermeldungen protokolliert.
- **Versionsanzeige**: Doppeltes "v"-Praefix in der Footer-Versionsanzeige entfernt.

---

## v1.1.0 — 2026-04-05

### Neue Funktionen

- **Tagesordnungsimport-Redesign**: Eingabe und Korrektur in einem Live-Zwei-Panel-Workflow kombiniert mit Klassifizierungspillen, Formaterkennung und Diff-Ansicht mit ebenenuebergreifender Verschiebungserkennung.
- **Tagesordnungs-Zeitstempel**: Zeitstempel werden erfasst, wenn Tagesordnungspunkte waehrend einer Sitzung betreten und verlassen werden.
- **Abstimmungsquittungen**: Abstimmungsentscheidungen werden bei der Ueberpruefung geheimer Abstimmungsquittungen angezeigt. Dialog "Meine Quittungen" zur Live-Sitzungsansicht hinzugefuegt.
- **Sitzungserstellungs-Assistent**: Mehrstufiger Assistent fuer die Sitzungserstellung mit Tagesordnungs- und Teilnehmerimport.
- **In-App-Dokumentation**: Eingebettete lokalisierte Benutzerdokumentation mit Suche, Medienvarianten und kontextsensitiver Hilfe.
- **SVG-Logo**: Text-Titel im Header durch SVG-Logo ersetzt.
- **Internationalisierung**: Paraglide-Uebersetzungen in allen Svelte-Komponenten aktiviert.
- **CI-Pipeline**: Dreistufiges Dockerfile mit Screenshot-Generierung, Upgrade auf Go 1.26.1.

### Verbesserungen

- **Moderationsseite**: Wiederverwendbare Komponenten extrahiert (AttendeeRow, VoteCard, AgendaPointCard, SpeakersSection, VotesPanelSection).
- **Admin-Seiten**: Admin-Seitenlayouts mit gemeinsamen AppCard- und DataTable-Komponenten aufgeraeumt.
- **QR-Codes**: QR-Code-Seiten in Inline-Dialog-Modale umgewandelt.
- **Fehlerbehebungen**: 10 Eintraege aus IMPROVEMENTS.md behoben, darunter Svelte-5-Reaktivitaetszyklen, Redelisten-Probleme und UI-Inkonsistenzen.

---

## v1.0.0 — 2026-04-05

### Erstveroeffentlichung

Open Caucus — ein Konferenz- und Gremienverwaltungstool.

- **Gremienverwaltung**: Gremien erstellen und konfigurieren mit Mitgliedsrollen (Vorsitz, Mitglied).
- **Sitzungslebenszyklus**: Sitzungen erstellen, Tagesordnungspunkte mit Unterpunkten verwalten und Live-Sitzungen durchfuehren.
- **Redeliste**: Echtzeit-Redeliste mit SSE-Live-Updates, Geschlechterquotierung, Prioritaetsschaltern und Moderationszuweisung.
- **Abstimmungen**: Normalisierter Abstimmungslebenszyklus auf Tagesordnungspunkten mit geheimen Abstimmungsquittungen und oeffentlicher Verifizierung.
- **Teilnehmerverwaltung**: Beitrittsablauf mit QR-Codes, Gastsitzungen und Teilnehmer-Self-Service.
- **Authentifizierung**: Kontobasierter Login mit Admin- und Benutzerrollen. OAuth/OIDC-Unterstuetzung mit Anbieter-Gating und automatischer Gremienmitgliedschafts-Synchronisation.
- **Tagesordnungsimport**: Tagesordnung aus Text importieren mit Korrektur und Diff-Vorschau.
- **Anhaenge**: Tagesordnungspunkt-Anhaenge hochladen und verwalten.
- **Internationalisierung**: Lokalisierungsbewusstes Routing mit englischen und deutschen Uebersetzungen.
- **Echtzeit-Updates**: SSE-basierte Live-Updates fuer Redeliste, Abstimmungen und Sitzungsstatus.
- **Anpassbare Panels**: Responsive Layout mit anpassbaren Panels in der Moderationsansicht.
- **Dokumentationserfassung**: Skriptgesteuerte Screenshot- und GIF-Generierung fuer die Dokumentation.
- **SPA-Architektur**: SvelteKit-Frontend mit Connect (gRPC-web) API, vollstaendig vom alten HTMX-Layer entkoppelt.
