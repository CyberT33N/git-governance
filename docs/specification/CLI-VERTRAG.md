# CLI-Vertrag für `git-governance`

## 1. Aufruf und Benennung

Die Release-Binary heißt:

```text
git-governance
```

Git erkennt ausführbare Dateien nach dem Muster `git-<name>` als Subcommand. Deshalb sind beide Formen äquivalent:

```text
git governance branch create
git-governance branch create
```

Die primäre Dokumentation verwendet `git governance ...`.

Die alten Namen `mkbranch` und `mkcommit` werden nicht zur Zieloberfläche. Ein einzelnes Root-Kommando ist notwendig, weil Branches, Commits, Validierung, Konfiguration und Workflows dieselben Domain-Objekte und dieselbe Release-Version verwenden. Fachlich getrennte Use Cases bleiben getrennte Subcommands.

## 2. Globale Optionen

```text
--interactive auto|always|never   Standard: auto
--output human|json              Standard: human
--quiet                          nur notwendige Ausgabe
--color auto|always|never        Standard: auto
--accessible                     vereinfachte Screenreader-Oberfläche
--remote <name>                  Standard: origin
--repo <path>                    Standard: aktuelles Verzeichnis
--config <path>                  explizite Konfigurationsdatei
--quality-config <path>          explizite Repository-Quality-Gate-Datei
--dry-run                        Plan anzeigen, nichts mutieren
--yes                            bestätigbare Schritte freigeben
--timeout <duration>             Grenze für externe Prozesse
```

Regeln:

- `auto` startet Formulare nur bei vorhandenem TTY und fehlenden Pflichtwerten.
- `never` liest niemals interaktiv; fehlende Werte sind ein Nutzungsfehler.
- `always` scheitert klar, wenn kein TTY verfügbar ist.
- `always` ist mit `--output=json` unzulässig, weil JSON keine Prompts enthält.
- `--yes` ersetzt keine fehlenden fachlichen Werte.
- `--quiet` verändert weder Interaktion noch Validierung.
- `--color=auto` verwendet Farbe nur bei Terminalausgabe; `always` erzwingt
  ANSI-Farbe und `never` verwendet reine Textausgabe.
- Im JSON-Modus sind Prompts verboten; bei `--interactive=auto` verhält sich JSON deshalb wie `never`.
- Secrets werden weder über Flags noch über diese Konfigurationsdatei verwaltet.

`--quality-config` ist keine Spracheinstellung. Es zeigt auf einen
repository-lokalen, explizit vertrauenswürdigen JSON-Vertrag aus ausführbaren
Command-/Argumentarrays. Fehlt die Datei, lautet das Ergebnis
`qualityStatus=unconfigured`; es wird niemals als bestandener Build oder Lint
ausgegeben.

## 3. Command Tree

```text
git governance
├── branch
│   ├── list
│   ├── create
│   ├── validate
│   └── sync-base
├── commit
│   ├── create
│   └── validate
├── workflow
│   ├── ticket
│   │   ├── start
│   │   └── publish
│   ├── hotfix
│   │   ├── start
│   │   ├── publish
│   │   └── propagate
│   └── release
│       ├── cut
│       ├── stabilize
│       ├── publish-stabilization
│       ├── promote
│       ├── backmerge
│       └── support
├── cleanup
├── validate
│   └── pre-push
├── config
│   └── key
│       ├── list
│       ├── add
│       ├── remove
│       └── set-default
├── policy
│   └── describe
├── completion
└── doctor
```

## 4. `branch list`

Zeigt alle Branch-Familien einschließlich Shared Lines und governance-gebundener Linien:

- `main`
- `develop`
- `release`
- `support`
- `feature`
- `fix`
- `docs`
- `refactor`
- `chore`
- `test`
- `perf`
- `hotfix`
- `scratch`

Jeder Eintrag enthält:

- Rolle
- Naming-Form
- zulässige Startbasis
- typisches PR-Ziel
- Protection-/Rewrite-Regel
- ob die Familie über `branch create` oder einen Workflow erzeugt wird

`branch list` ist die vollständige Informationsoberfläche. `branch create` zeigt nur auswählbare Familien für den konkreten Kontext und erklärt, warum andere Familien nicht direkt erzeugt werden dürfen.

## 5. `branch create`

### 5.1 Zweck

Erzeugt genau einen Branch aus einer explizit bestimmten Basis. Das Kommando validiert und mutiert Git; es enthält keinen Ticket-bis-PR-Gesamtworkflow.

### 5.2 Optionen

```text
--family feature|fix|docs|refactor|chore|test|perf|scratch
--key <KEY>
--ticket <NUMBER>
--slug <kebab-case>
--base <remote-ref>
--switch                        Standard: true
```

Regeln:

- Fehlt `--family` interaktiv, erscheint eine Auswahl mit Erklärung jeder Familie.
- Für reguläre Ticket-Familien ist die Standardbasis `<remote>/develop`.
- Vor einer echten Erstellung prüft das Kommando nach `fetch --prune`, ob
  bereits ein lokaler oder ausgewählter Remote-Tracking-Branch für dasselbe
  Ticket existiert. Ein zweiter regulärer offizieller Ticket-Branch wird
  abgewiesen.
- `hotfix` verlangt die real betroffene Basis und wird über `workflow hotfix start` erzeugt.
- `scratch` wird aus einem lokalen offiziellen Ticket-Branch desselben Tickets erzeugt; bei direkter Auswahl wird diese Basis abgefragt.
- `scratch` akzeptiert keine Remote-Tracking-Referenz, Shared Line, andere Scratch-Basis oder ticketfremde Basis.
- `release` und `support` verweisen auf governance-gebundene Workflow-Kommandos.
- `main` und `develop` sind keine auswählbaren Arbeitsbranches.
- Das Kommando führt nie `git add`, Commit, Amend oder Force Push aus.

### 5.3 Nicht-interaktives Beispiel

```text
git governance branch create \
  --interactive never \
  --family feature \
  --key ABC \
  --ticket 123 \
  --slug add-export-button \
  --output json
```

Der generierte Name ist:

```text
feature/ABC-123-add-export-button
```

### 5.4 Mutationsplan

Vor Bestätigung oder bei `--dry-run` wird ein Plan angezeigt:

```text
Remote aktualisieren: git fetch --prune origin
Basis prüfen: refs/remotes/origin/develop
Branch erzeugen: feature/ABC-123-add-export-button
Startpunkt: origin/develop
Arbeitsbranch wechseln: ja
```

## 6. `branch validate`

```text
git governance branch validate [<branch-name>]
```

Ohne Argument wird der aktuelle Branch verwendet. Das Kommando prüft:

- vollständige Branch-Grammatik
- Value-Object-Regeln
- `git check-ref-format --branch`
- Family-spezifische Regeln
- optional Key-Policy/Bundle
- bei vorhandenem Repository den zulässigen Arbeitskontext

Es mutiert nichts und eignet sich für lokale Diagnose und CI.

## 7. `branch sync-base`

### 7.1 Zweck

Stellt fest, ob der aktuelle offizielle Arbeitsbranch Commits seiner tatsächlichen Zielbasis vermisst. Das Kommando ersetzt keine Merge Queue und führt keinen blinden Rebase aus.

```text
git governance branch sync-base \
  --strategy check|auto|rebase|merge
```

### 7.2 Entscheidungslogik

1. aktuellen Branch parsen
2. tatsächliche Zielbasis bestimmen
3. sauberen Arbeitsbaum prüfen
4. `git fetch --prune <remote>`
5. Publication State prüfen
6. fehlende Basis-Commits bestimmen
7. Policy anwenden

| Zustand | Ergebnis |
|---|---|
| keine fehlenden Basis-Commits | `BASE_UP_TO_DATE`, keine Mutation |
| unveröffentlicht und Basisdelta vorhanden | Rebase zulässig |
| veröffentlicht und Basisdelta vorhanden | Rebase verboten; optional kontrollierter Merge |
| Publication State unbekannt | History Rewrite blockieren |
| Shared Line oder Scratch | eigener Family-Vertrag |

`auto` bedeutet nicht „immer mutieren“:

- unveröffentlicht: nur bei Delta rebasen
- veröffentlicht: ohne explizite Merge-Freigabe nur Handlungsplan ausgeben

Nach einer Mutation laufen Governance-Checks und konfigurierte Quality Checks erneut. Schlägt ein Rebase konfliktbedingt fehl, bleibt Git im normalen Rebase-Zustand; die CLI zeigt `--continue`-, `--abort`- und Diagnoseanweisungen und versteckt den Zustand nicht.

## 8. `commit create`

### 8.1 Optionen

```text
--type build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test
--ticket <KEY-NUMBER>
--subject <text>
--body <text>
--breaking
--breaking-description <text>
--footer <token=value>           wiederholbar
--stage <path>                   wiederholbar
--push                           Standard: false
```

### 8.2 Defaults und Ableitungen

- Das Ticket wird auf einem Ticket-Branch aus dem Branch-Namen abgeleitet.
- Ein explizites `--ticket` muss exakt zum Branch passen.
- Der Commit-Typ wird aus der Branch-Familie vorgeschlagen, aber nicht blind erzwungen.
- `feature` schlägt `feat`, `fix` und `hotfix` schlagen `fix` vor.
- `docs`, `refactor`, `chore`, `test` und `perf` schlagen den gleichnamigen Typ vor.
- Das Kommando prüft, ob Änderungen gestaged sind.
- Ohne `--stage` wird niemals automatisch `git add .` ausgeführt.
- `--stage` akzeptiert explizite Pfade und zeigt sie vor der Mutation.
- `--push` ist optional und läuft durch dieselbe Pre-Push-Validierung wie Lefthook.

### 8.3 Breaking Changes

Bei `--breaking` erzeugt die UI standardmäßig:

```text
feat(ABC-123)!: replace the export contract

BREAKING CHANGE: clients must consume the new resource envelope.
```

Der Benutzer erhält eine Erklärung:

- Breaking bedeutet inkompatible öffentliche Vertragsänderung.
- Der Marker darf nicht für interne Refactors missbraucht werden.
- Die Beschreibung muss konkrete Migrationsauswirkung nennen.

### 8.4 Amend und Force Push

`commit create` bietet kein Amend-Flag. Vor dem ersten Push wäre ein lokales Amend gemäß Referenz-Governance zwar grundsätzlich zulässig, ist aber kein notwendiger Produkt-Use-Case. Nach dem ersten Push ist Amend als Routine verboten. Force Push wird von keinem Kommando angeboten.

## 9. `commit validate`

```text
git governance commit validate --message-file <path>
git governance commit validate --message <text>
```

Prüfungen:

- Header-Grammatik
- Commit-Typ
- Ticket-ID
- Beschreibung
- Body-/Footer-Struktur
- Breaking-Change-Semantik
- Ticketkonsistenz zum aktuellen Branch
- Shared-Line-Regeln
- optionale Key-Policy

Für `commit-msg` wird immer `--message-file` verwendet. Die Datei wird begrenzt gelesen; NUL und unzulässige Kontrollzeichen werden abgewiesen.

## 10. `workflow ticket start`

### 10.1 Zweck

Startet reguläre Ticket-Arbeit und endet auf dem offiziellen oder optionalen Scratch-Branch.

```text
git governance workflow ticket start \
  --family feature \
  --key ABC \
  --ticket 123 \
  --slug add-export-button \
  --scratch
```

### 10.2 Ablauf

1. Repository und Git-Version prüfen.
2. Arbeitsbaum und laufende Git-Operationen prüfen.
3. Ticket-Eingaben validieren.
4. `git fetch --prune origin`.
5. offiziellen Branch direkt von `origin/develop` erzeugen.
6. optional Scratch-Frage mit Erklärung anzeigen.
7. bei Zustimmung `scratch/<ticket>-<scratch-slug>` vom offiziellen Branch erzeugen.
8. auf dem gewählten Branch enden.

`--scratch` erstellt ausdrücklich eine private Exploration. Ohne Flag fragt der
interaktive Modus nach; nicht-interaktiv wird ohne Flag kein Scratch-Branch
angelegt.

### 10.3 Scratch-Erklärung in der UI

Die UI muss vor der Auswahl sinngemäß anzeigen:

```text
Scratch-Branches sind private, kurzlebige Explorationslinien.
Verwende Scratch nur, wenn Lösungsweg oder Experiment unsicher sind.
Erstelle keinen Pull Request aus Scratch und teile ihn nicht als offiziellen
Arbeitsbranch. Übernimm stabile Ergebnisse später kontrolliert per Squash
oder Cherry-Pick in den offiziellen Ticket-Branch.
```

## 11. `workflow ticket publish`

Dieses Kommando wird nach Entwicklung und lokalen Tests aufgerufen. Es ist kein automatisch fortlaufender Teil von `ticket start`.

```text
git governance workflow ticket publish \
  --push --draft
```

Ablauf:

1. offiziellen Ticket-Branch und sauberen Zustand prüfen
2. Branch- und Commit-Serie validieren
3. projektdefinierte Quality Checks ausführen
4. Basisfrische prüfen
5. bei unveröffentlichtem Branch und Basisdelta nach Bestätigung rebasen
6. nach einem Rebase Branch-/Policy-Prüfung, Commit-Serie und Quality Gates erneut ausführen
7. ersten Push ausführen
8. PR-Payload mit Head, Base `develop`, Ticket und vorgeschlagenem Titel erzeugen

Ohne Provider-Adapter wird kein Hosting-API-Aufruf erfunden. Die JSON-Ausgabe ist eine stabile Übergabeoberfläche für GitHub-, GitLab-, Bitbucket- oder andere Adapter.

## 12. `workflow hotfix start`

Pflichtoptionen:

```text
--key <KEY>
--ticket <NUMBER>
--slug <slug>
--affected-line main|release/<semver>|support/<major.minor>
```

Ablauf:

1. betroffene Linie validieren
2. `fetch --prune`
3. Remote-Linie und sauberen Arbeitsbaum prüfen
4. `hotfix/<ticket>-<slug>` direkt von der betroffenen Remote-Linie erzeugen
5. Ziel-PR auf dieselbe Linie festlegen

Ein Hotfix startet nie automatisch von `develop`.

### 12.1 `workflow hotfix publish`

```text
git governance workflow hotfix publish \
  --affected-line main|release/<semver>|support/<major.minor> \
  --push
```

Der Befehl verlangt die tatsächlich betroffene Linie erneut, validiert den
Hotfix gegen dieselbe Basis und erzeugt den PR-Intent auf genau diese Linie.
Ein Hotfix wird niemals stillschweigend nach `develop` umgeleitet.

### 12.2 `workflow hotfix propagate`

```text
git governance workflow hotfix propagate \
  --target-line main|develop|release/<semver>|support/<major.minor> \
  --commit <sha> \
  --push
```

Der Befehl erzeugt einen kontrollierten `fix/*`-Branch aus der Ziel-Linie,
führt `git cherry-pick -x <sha>` aus und bereitet den PR gegen diese Ziel-Linie
vor. Damit bleibt die Herkunft eines Forward- oder Backports nachweisbar.

## 13. Release-Kommandos

### 13.1 `workflow release cut`

```text
git governance workflow release cut --version 2.8.0
```

Das Kommando:

- verlangt eine explizite Governance-Bestätigung
- aktualisiert `origin/develop` per Fetch
- erzeugt `release/2.8.0` direkt von `origin/develop`
- pusht nur bei expliziter Freigabe
- erklärt die danach erlaubte begrenzte Stabilisierung

### 13.2 `workflow release stabilize`

```text
git governance workflow release stabilize \
  --release release/<semver> \
  --kind blocker|docs|release-prep \
  --key <KEY> --ticket <NUMBER> --slug <kebab-case>
```

Nur die drei genannten Kategorien sind auf einer eingefrorenen Release-Linie
zulässig. Neue Features, allgemeine Refactors und themenfremde Tickets besitzen
keine auswählbare Stabilisierungskategorie.

### 13.3 `workflow release publish-stabilization`

Dieser Befehl validiert einen Stabilisierung-Branch gegen
`origin/release/<semver>` und erzeugt seinen PR-Intent auf dieselbe
Release-Linie.

### 13.4 `workflow release promote`

Dieser Befehl erzeugt den providerneutralen PR-Intent:

```text
release/<semver> -> main
```

Tagging und Artefakterstellung folgen erst nach dem geschützten Merge in der
Release-Pipeline. Der CI-Workflow erzeugt `v<semver>` direkt auf dem
Merge-Commit und startet anschließend den Artefaktworkflow für genau diesen
unveränderlichen Tag.

### 13.5 `workflow release backmerge`

Erzeugt keine stillen Direktcommits. Das Kommando validiert Release- und Zielzustand und erzeugt providerneutrale PR-Daten für:

```text
release/<semver> -> develop
```

Die Freigabe nach `main`, Tagging und Artefakterstellung bleiben Release-/CI-Verantwortung.

### 13.6 `workflow release support`

`support/<major.minor>` darf nur erzeugt werden, wenn die aktuell gefetchte
`origin/main`-Revision einen passenden `v<major.minor.patch>`-Release-Tag
trägt.

### 13.7 `workflow cleanup`

Der Cleanup-Workflow löscht nach ausdrücklicher Bestätigung abgeschlossene
Ticket-, Scratch-, Hotfix- oder Release-Branches lokal und optional remote.
`main`, `develop` und aktive `support/*`-Linien bleiben ausgeschlossen.

## 14. `validate pre-push`

Dieses Kommando ist die Lefthook- und manuelle Pre-Push-Oberfläche.

```text
git governance validate pre-push \
  --remote origin
```

Es liest die von Git gelieferte Ref-Liste begrenzt von stdin und prüft:

- jede vierfeldrige Git-Ref-Aktualisierung statt nur den aktuell ausgecheckten Branch
- Ziel-Branch-Grammatik und Key-Policy
- Shared-Line-Push-Verbot
- Commit-Ticket-Konsistenz
- Löschungen, nicht-fast-forward-/Rewrite-Versuche und Mehrfach-Updates
- Bundle-Präsenz und -Frische, sobald der Bundle-Adapter aktiv ist
- Basislinien-Frische vor dem ersten Push

Der Validator führt nie selbst Rebase oder Merge aus. Er blockiert mit einer konkreten, policy-konformen Handlungsanweisung.

## 15. Konfigurationskommandos

```text
git governance config key add PLATFORM2
git governance config key list
git governance config key set-default PLATFORM2
git governance config key remove PLATFORM2
```

Regeln:

- nur syntaktisch gültige Keys werden gespeichert
- Speicherung ist dedupliziert und atomar
- ein gespeicherter Key gilt nicht automatisch als Registry-zugelassen
- Ticketnummern werden nicht als globaler Default gespeichert
- Commits leiten das Ticket aus dem aktuellen Branch ab

## 16. `policy describe`

Gibt die aktive ausführbare Policy versioniert aus:

```text
git governance policy describe --output json
```

Enthalten sind:

- Policy-Schema-Version
- Branch-Familien
- Commit-Typen
- Regex-/Grammatik-IDs
- aktive Key-Policy (`syntax-only` oder `bundle`)
- technische Limits
- Fehlercodes

Dokumentations- und Conformance-Tests verwenden diese Ausgabe, damit keine zweite Regex-Wahrheit in Hooks oder Beispielen entsteht.

## 17. `doctor`

Read-only-Diagnose:

- unterstütztes Betriebssystem und Architektur
- Git vorhanden und Mindestversion erfüllt
- Repository erkannt
- Remote vorhanden, ohne dessen URL in der Human-Ausgabe offenzulegen
- Benutzerkonfiguration lesbar
- Lefthook vorhanden
- Lefthook-Konfiguration vorhanden
- Policy-Bundle-Status, wenn aktiviert
- keine laufende Merge-/Rebase-/Cherry-Pick-Operation

`doctor` installiert, repariert oder mutiert nichts ohne ein separates explizites Kommando.

## 18. Human- und JSON-Ausgabe

### 18.1 Human

Fehlerdarstellung:

```text
Fehler [COMMIT_TICKET_MISMATCH]

Tatsächlicher Wert:
  ABC-124

Was ist falsch?
  Der Commit verwendet ABC-124, der aktuelle Branch gehört zu ABC-123.

Wie muss es sein?
  Alle Commits eines offiziellen Ticket-Branches verwenden dessen Ticket-ID.

Gültiges Beispiel:
  feat(ABC-123): add export button

Behebung:
  Verwende ABC-123 oder wechsle auf den zum Commit gehörenden Branch.
```

### 18.2 JSON

```json
{
  "schemaVersion": 1,
  "ok": false,
  "error": {
    "code": "COMMIT_TICKET_MISMATCH",
    "category": "governance",
    "field": "ticket",
    "actual": "ABC-124",
    "expected": "ABC-123",
    "rule": "commit ticket must equal branch ticket",
    "example": "feat(ABC-123): add export button",
    "remediation": "use ABC-123 or switch branches"
  }
}
```

JSON-Feldnamen und Exitcodes sind öffentliche Verträge und werden kompatibel versioniert.

## 19. Interne Komposition

Delivery-Adapter sammeln Eingaben und erzeugen Commands. Workflows rufen Application Services direkt auf:

```text
Cobra/Huh
→ StartTicketWorkflow
  → FetchRemote
  → CreateBranch
  → optional CreateScratchBranch
→ Reporter
```

Nicht zulässig:

```text
workflow command
→ startet git-governance branch create als Kindprozess
→ parst dessen Textausgabe
```

Nur externe Consumer und Automation verwenden die CLI-Oberfläche.

## 20. Übernahme aus dem bisherigen Tool

| Bestehende Fähigkeit | Zielentscheidung |
|---|---|
| interaktive Branch-Auswahl | übernehmen, aber vollständige kanonische Taxonomie |
| interaktive Commit-Typ-Auswahl | übernehmen und vervollständigen |
| Ticket-Key-Historie | als OS-konforme Benutzerpräferenz übernehmen |
| Ticketnummer-Eingabe | übernehmen, aber nicht global wiederverwenden |
| Branch-Slug-Eingabe | übernehmen mit strengerem kebab-case |
| Bestätigung vor Mutation | übernehmen plus `--dry-run` |
| Wechsel auf neuen Branch | übernehmen |
| optionaler Push nach Commit | nur explizit und durch Pre-Push-Validierung |
| Checkout und Pull von `develop` | durch Fetch und direkte Basisreferenz ersetzen |
| Dev-/User-Suffixe im Branchnamen | verwerfen |
| automatischer Initial Commit | verwerfen |
| eigene Hook-Kopierskripte | durch Lefthook-Standard ersetzen |
| parallele PowerShell-/Shell-Logik | vollständig verwerfen |
