---
title-en: Agenda Management and Import
title-de: Tagesordnung und Import
---

# Tagesordnung und Import

Auf dieser Seite verwaltest du Tagesordnungspunkte während der Sitzung und importierst eine vorbereitete Tagesordnung sicher.

## Für wen diese Seite gedacht ist

Sitzungsleitungen, die die Tagesordnung organisiert und im Einklang mit der aktuellen Diskussion halten.

## Bevor du loslegst

1. Öffne die Meeting-Moderationsseite und wechsle zum **Agenda**-Tab.
2. Entscheide, ob du manuelle Änderungen, importbasierte Aktualisierungen oder beides benötigst.
3. Falls du importierst, bereite deine Tagesordnung als Text vor (z. B. Text mit `#`-Überschriften oder nummerierten Zeilen).

## Layout auf Desktop vs. Mobil

- Desktop:
  Du kannst die Tagesordnungssteuerung sichtbar halten und gleichzeitig andere Moderationsbereiche einsehen.
- Mobil:
  Bearbeite zuerst den **Agenda**-Tab und scrolle dann zu anderen Moderationsbereichen.

## Tagesordnung bearbeiten

Verwende dies für die normale Tagesordnungspflege während einer Sitzung.

### Schritt für Schritt

1. Tagesordnungspunkte hinzufügen:
   - Gib unter **Add Agenda Point** einen Titel ein.
   - Wähle optional einen übergeordneten Punkt unter **Parent (optional)**.
   - Klicke auf **Add**.
2. Punkte umsortieren:
   - Nutze **Move up** und **Move down** auf den Tagesordnungskarten.
3. Aktuellen Diskussionspunkt aktivieren:
   - Klicke auf **Activate agenda point** beim richtigen Eintrag.
   - Die Eintrittszeit wird erfasst und neben jedem Punkt in der Seitenleiste angezeigt. Wenn du zu einem anderen Punkt wechselst, wird auch die Dauer erfasst.
4. Punkt-spezifische Aktionen öffnen:
   - Klicke auf **Open tools** auf der Tagesordnungskarte.
5. Veraltete Punkte entfernen:
   - Klicke auf **Delete agenda point** und bestätige.

### Worauf du nach Änderungen achten solltest

- Der aktive Punkt stimmt mit der tatsächlichen Diskussion überein.
- Die Eltern-Kind-Struktur ist weiterhin korrekt.
- Auf dem Handy nach dem Zurückscrollen zum Anfang der Liste nochmals prüfen.

## Tagesordnung importieren (Übersicht)

Verwende dies, wenn du einen vorbereiteten Tagesordnungstext hast und kontrollierte Massenaktualisierungen möchtest.

1. Klicke auf **Import**, um **Import Agenda** zu öffnen.
2. Unter **Source** Text einfügen oder eine Datei hochladen, dann auf **Extract Agenda** klicken.
3. Unter **Correction** die erkannten Zeilen prüfen und Zeilen anklicken, um sie als `Ignore`, `Heading` (Hauptpunkt) oder `Subheading` (Unterpunkt) zu setzen.
4. Auf **Generate Diff** klicken.
5. Unter **Diff** (Änderungsvorschau) alle Änderungen sorgfältig prüfen.
6. Auf **Accept** klicken, um anzuwenden, oder **Deny**, um abzubrechen.
7. Nach dem Anwenden prüfen, ob der aktive Punkt noch korrekt ist.

## Tagesordnung importieren (Funktionsweise)

### Welche Textformate funktionieren

1. Text mit `#`- und `##`-Überschriften:
   - Zeilen, die mit `#` beginnen, werden als Hauptpunkte behandelt.
   - Zeilen, die mit `##` beginnen, werden als Unterpunkte behandelt.
   - Bei manchen Tagesordnungsstilen ist die erste `#`-Zeile nur ein Titel; in diesem Fall konzentriert sich der Importer auf die niedrigeren Überschriftsebenen.
2. Nummerierter Text:
   - Der Importer versteht gängige Formate wie `1`, `1.1` oder `TOP1`.
   - Hauptnummern werden zu Tagesordnungspunkten, verschachtelte Nummern zu Unterpunkten.
3. Eingerückter Text:
   - Weniger eingerückte Zeilen werden zu Hauptpunkten.
   - Stärker eingerückte Zeilen werden zu Unterpunkten.
4. Wenn eine Zeile nicht zur gewählten Struktur passt, kann sie im Korrekturschritt auf `Ignore` gesetzt werden.

### Korrekturoptionen

- `Ignore`: diese Zeile überspringen.
- `Heading`: als Haupttagesordnungspunkt verwenden.
- `Subheading`: als Unterpunkt unter dem letzten Hauptpunkt verwenden.
- Eine `Subheading`-Zeile muss nach einer `Heading`-Zeile stehen.

### Logik der Änderungsvorschau

1. Der Importer erstellt einen Vorschlag aus den korrigierten Zeilen.
2. Er vergleicht diesen Vorschlag mit deiner aktuellen Tagesordnung:
   - Zuerst werden exakte Titelübereinstimmungen gesucht.
   - Falls das fehlschlägt, werden wahrscheinliche Übereinstimmungen basierend auf ähnlichem Wortlaut und Position gesucht.
3. Jede Zeile in der Änderungsvorschau wird markiert als:
   - `insert`: neuer Punkt wird hinzugefügt
   - `delete`: bestehender Punkt wird entfernt
   - `move`: Punkt bleibt, ändert aber Position/Elternelement
   - `rename`: Punkt behält Position, aber der Titel ändert sich
   - `unchanged`: keine Änderung
4. Diese Vorschau zeigt alle Änderungen klar an, bevor du sie anwendest.

### Sicherheitsprüfung vor dem Anwenden

- Die App prüft, ob sich die Tagesordnung geändert hat, während du die Vorschau überprüft hast.
- Falls sie sich geändert hat, wird eine Warnung angezeigt, und du musst die aktualisierte Vorschau erneut prüfen, bevor du sie anwendest.

## Falls etwas schiefgeht

- Import meldet, dass die Quelle leer/zu groß/nicht parsebar ist:
  Bereinige den Quelltext und versuche es erneut mit einer kleineren, strukturierten Eingabe.
- Korrekturschritt schlägt fehl:
  Stelle sicher, dass mindestens eine Zeile als `Heading` (Hauptpunkt) markiert ist und vermeide `Subheading` (Unterpunkt) vor einem Hauptpunkt.
- Diff warnt, dass sich die Tagesordnung während der Überprüfung geändert hat:
  Prüfe die aktualisierte Änderungsvorschau erneut, bevor du akzeptierst.
- Importergebnis sieht falsch aus:
  Nutze **Deny**, passe Quelle/Korrekturen an und generiere die Vorschau erneut.

## Tagesordnungsrouten

Verwende diese Übersicht, wenn die Moderationshilfe diese Seite unter `agenda-routes` öffnet:

- **Agenda**-Tab: tägliche Punkteverwaltung.
- **Add Agenda Point**: manueller Erstellungsprozess.
- **Import Agenda**: `Source` -> `Correction` -> `Diff` Importschritte.
- **Activate agenda point**: setzt den aktuell diskutierten Punkt.

## Wie es weitergeht

Weiter mit [Teilnehmende Signup und Recovery](/docs/03-chairperson/04-attendees-signup-and-recovery) und [Redeliste Moderation und Quotierung](/docs/03-chairperson/05-speakers-moderator-and-quotation), um die Live-Sitzung zu leiten.
