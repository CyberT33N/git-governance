# ADR-0001: Native Go-CLI als gemeinsame Git-Governance-Anwendung

- Status: angenommen und im lokalen Produktkern implementiert
- Datum: 2026-07-10
- Entscheidungsart: neue Zielarchitektur
- Produktname: `git-governance`
- Primärer Aufruf: `git governance ...`
- Direkter Aufruf: `git-governance ...`

## 1. Ergebnis

Die Zielarchitektur ist ein modularer Go-Monolith als einzelne native Binary. Branches, Commits, Workflows und alle zugehörigen syntaktischen Validierungen verwenden denselben Domain-Kern. Lefthook bleibt die verbindlich vorgegebene dünne lokale Hook-Orchestrierung und ruft Validierungs-Use-Cases derselben Binary auf. CI verwendet dieselben maschinenlesbaren Validierungsoberflächen; serverseitige Branch Protection bleibt die bindende letzte Instanz.

Damit entstehen weder parallele PowerShell-/Shell-Wahrheiten noch Regex-Duplikate in Hook-Dateien.

## 2. Eigenständiger Produktvertrag

Diese Architekturentscheidung ist die vollständige lokale Autorität für
`git-governance`. Sie definiert die Produktgrenzen, die Branch- und
Commit-Konventionen, den Workflow, die Installationsform und die
Verifikationsanforderungen selbst. Für die Nutzung oder Weiterentwicklung
werden keine externen Regeldateien benötigt.

Der Produktvertrag setzt insbesondere fest:

- eine einzelne native Go-Binary für Windows, macOS und Linux,
- einen gemeinsamen Domain-Kern für Branches, Commits und Workflows,
- die vollständige Branch-Taxonomie von `main` bis `scratch/*`,
- Conventional Commits mit verpflichtendem Ticket-Scope,
- `fetch --prune` und direkte Remote-Basen statt lokaler Pull-Workarounds,
- eine veröffentlichungsabhängige Rebase- und Rewrite-Policy,
- Lefthook als dünnen lokalen Hook-Runner,
- CI und Remote Branch Protection als bindende Durchsetzungsinstanzen,
- eine isolierte Syntax-Key-Policy mit später austauschbarem Bundle-Adapter,
- vorgebaute, signierte Release-Artefakte statt Compiler auf Endgeräten.

Der bestehende Shell-/PowerShell-Code wurde ausschließlich als Inventar
vorhandener Fähigkeiten betrachtet. Seine Benennung, Struktur, Validierung,
Bedienoberfläche und Installationslogik sind keine Zielautorität.

## 3. Klassifikation

- Primäre Projektklasse: Cross-Platform CLI
- Sekundäre Klassen: Developer Tooling, Git-Automation, lokale Policy-Validierung
- Business Capability: regelkonforme Git-Arbeit vom Ticket über Branch und Commit bis zur PR-Vorbereitung
- Ausführung: Windows, macOS und Linux, interaktiv sowie nicht-interaktiv
- Delivery: signierte native Binary je Betriebssystem und Architektur
- Zustände: Git-Repository, Benutzerpräferenzen und zukünftig ein lokales Policy-Bundle
- Kritische Seiteneffekte: `fetch`, Branch-Erstellung, `switch`, Commit, Push sowie konditional Rebase oder Merge
- Nicht-Ziele: Ersatz für Git, Git-Hosting, CI, serverseitige Branch Protection oder die Policy Registry

## 4. Korrekturen an den Ausgangsannahmen

### 4.1 `develop` muss nicht ausgecheckt und gepullt werden

Für einen neuen regulären Ticket-Branch ist folgender Ablauf kanonisch:

```text
git fetch --prune origin
git switch -c feature/ABC-123-add-export-button origin/develop
```

`git fetch` aktualisiert die Remote-Tracking-Referenz `origin/develop`. Der Branch kann direkt davon erzeugt werden. Ein vorheriges `git switch develop` mit `git pull` mutiert unnötig den lokalen `develop` und scheitert leichter bei lokalem Drift.

### 4.2 Rebase ist kein allgemeiner späterer Workflow

Ein Rebase ist nur zulässig, wenn alle Bedingungen erfüllt sind:

1. Der offizielle Arbeitsbranch wurde noch nie veröffentlicht.
2. `git fetch --prune origin` war erfolgreich.
3. Die tatsächliche Zielbasis ist bekannt.
4. `git log --oneline HEAD..origin/<target-base>` zeigt fehlende Basis-Commits.
5. Der Arbeitsbaum ist sauber.
6. Nach dem Rebase laufen die konfigurierten Validierungen erneut.

Nach dem ersten Push ist der offizielle Branch append-only. Dann sind Routine-Rebase, Amend und Force Push nicht zulässig; erforderliche Basisänderungen werden kontrolliert gemerged. Private `scratch/*`-Branches sind die ausdrücklich getrennte Ausnahme.

### 4.3 Interne Module rufen nicht die eigene CLI auf

Flags sind die externe Automatisierungsoberfläche. Intern orchestriert ein Workflow Application Services direkt im selben Prozess. Ein Selbstaufruf der Binary würde Prozessrekursion, String-/Exitcode-Kopplung und eine zweite Fehlerübersetzung erzeugen.

### 4.4 Nicht-interaktiv ist nicht „silent“

Die drei Achsen bleiben getrennt:

- Interaktion: `--interactive=auto|always|never`
- Ausgabeformat: `--output=human|json`
- Ausgabemenge: `--quiet`

`--interactive=never` verlangt alle notwendigen Werte als Argumente oder Flags. „Silent“ wäre mehrdeutig und darf nicht gleichzeitig Eingabemodus und Ausgabeformat bedeuten.

### 4.5 Ein durchgehender Ticket-bis-PR-Wizard ist fachlich falsch

Zwischen Branch-Start und Pull Request liegt die eigentliche Entwicklung mit Commits, Tests und Review-Vorbereitung. Deshalb existieren zwei atomare, wiederaufnehmbare Use Cases:

- `workflow ticket start`: aktuelle Basis holen, offiziellen Branch erzeugen, optional Scratch-Branch erzeugen und wechseln
- `workflow ticket publish`: lokalen Zustand validieren, Basisfrische prüfen, ersten Push vorbereiten oder ausführen und providerneutrale PR-Daten erzeugen

Es gibt keinen langlebigen Workflow-Prozess und keine versteckte Session-State-Machine.

## 5. Harte Gates

| Gate | Go-Binary + Lefthook | Zwei Go-Binaries + Lefthook | Remote-Policy-Service + Client | Lefthook-only |
|---|---|---|---|---|
| Windows/macOS/Linux | PASS | PASS | PASS | CONDITIONAL |
| Keine vorausgesetzte Sprachruntime | PASS | PASS | PASS | PASS für Lefthook selbst |
| Eine fachliche Wahrheit | PASS | PASS bei gemeinsamem Package | PASS | FAIL bei YAML/Scripts |
| Vollständige interaktive CLI | PASS | PASS | PASS | FAIL |
| Vollständige nicht-interaktive CLI | PASS | PASS | PASS | FAIL |
| Offline-fähiger `commit-msg`-Pfad | PASS | PASS | CONDITIONAL mit Cache | PASS nur als Runner |
| Kanonischer Lefthook-Standard | PASS | PASS | PASS | PASS |
| Branch-/Commit-/Workflow-Erstellung | PASS | PASS | PASS | FAIL |
| Signierbare native Delivery | PASS | PASS | PASS | nicht anwendbar auf fehlende Domain-App |

`Lefthook-only` wird vor der MCDA disqualifiziert: Lefthook kann beliebige Jobs ausführen, implementiert aber weder das Domain-Modell noch die geforderte Branch-, Commit-, Konfigurations- und Workflow-Oberfläche.

## 6. MCDA

Die Bewertung verwendet die Gewichte des Architecture Decision Blueprint.

| Kriterium | Gewicht | Eine Go-Binary | Zwei Go-Binaries | Remote-Service + Client |
|---|---:|---:|---:|---:|
| Domain- und Problem-Fit | 20 % | 9,5 | 9,2 | 8,8 |
| Security, Safety und Governance | 15 % | 9,0 | 8,8 | 8,0 |
| Korrektheit und Vertragsstärke | 12 % | 9,3 | 9,1 | 9,0 |
| Operability und Reliability | 12 % | 9,3 | 8,2 | 7,2 |
| Deployment und Portabilität | 10 % | 9,6 | 7,8 | 6,8 |
| Modularität und Wartbarkeit | 10 % | 9,2 | 9,6 | 9,2 |
| Performance und Ressourcen | 7 % | 9,0 | 9,1 | 7,5 |
| Verifikation und Tooling | 6 % | 9,4 | 9,2 | 8,8 |
| Ecosystem und Interoperabilität | 5 % | 9,0 | 8,8 | 8,3 |
| Longevity und Lock-in | 3 % | 9,1 | 8,7 | 8,0 |
| Absoluter Fit | 100 % | **92,8/100** | **88,7/100** | **82,1/100** |

Die normalisierten Anteile der drei zulässigen Optionen summieren sich auf 100 %:

| Rang | Option | Normalisierter Anteil | Einordnung |
|---:|---|---:|---|
| 1 | Eine native Go-Binary mit modularer Domain, Lefthook- und CI-Adaptern | **35 %** | stärkster Gesamtfit |
| 2 | Getrennte Go-Binaries für Workflow und Validator mit gemeinsamem Policy-Package | **34 %** | valide, aber doppelte Release- und Installationsfläche |
| 3 | Lokaler Go-Client mit Remote-Policy-Service und Offline-Cache | **31 %** | valide bei später nachgewiesener Zentralisierungsnotwendigkeit |

Der normalisierte Wert von 35 % ist kein Qualitätswert. Der absolute Fit der gewählten Architektur beträgt 92,8/100; die 35 % bilden nur den Anteil innerhalb einer Shortlist aus drei starken Kandidaten ab.

## 7. Sprach-, Runtime- und Frameworkentscheidung

### 7.1 Sprache

Go bleibt die richtige Sprache. Die projektbezogene MCDA priorisiert für diese Cross-Platform-CLI Go mit 44 %, Rust mit 32 % und TypeScript/Node mit 24 %. Die konkreten Anforderungen verstärken Go zusätzlich:

- einzelne native Binary ohne vorausgesetzte Laufzeit
- sehr gute Cross-Compilation für Windows, macOS und Linux
- schnelle Startzeit für Hooks
- Standardbibliothek für Prozesse, Pfade, JSON, Signale und Konfiguration
- klare Fehler- und Context-Semantik
- keine Notwendigkeit für `unsafe`, cgo oder Plugins

Zielstand:

- Sprachversion: Go 1.26
- gepinnte Build-Toolchain: Go 1.26.5
- cgo: deaktiviert, solange kein belegter Adapter es benötigt

Die Toolchain ist auf der Entwicklungsmaschine als Go 1.26.5 installiert.
Domain-, Adapter-, Application-, CLI- und lokale Git-Integrationstests sind
ausgeführt. Plattformübergreifende Release-Smoketests sowie Signierung bleiben
separate Release-Gates.

### 7.2 Delivery-Frameworks

- Command-Routing: Cobra
- Interaktive Terminal-Forms: Huh v2 hinter einem `Prompt`-Port
- Accessible Mode: explizit konfigurierbar und für Screenreader ohne TUI-Darstellung
- Konfiguration: Standardbibliothek und versioniertes JSON; kein globaler Viper-Zustand
- Git-Integration: installierte `git`-Binary über `exec.CommandContext` und Argumentarrays

Cobra und Huh bleiben Entry-Adapter. Domain und Application Layer importieren sie nicht.

## 8. Implementierte Struktur

```text
cmd/
  git-governance/
    main.go
internal/
  adapters/
    configfs/
    gitcli/
    quality/
    report/
    system/
    terminal/
  application/
    branch/
    commit/
    policy/
    port/
    workflow/
  bootstrap/
  domain/
    branch/
    commitmsg/
    problem/
    ticket/
  integration/
docs/
packaging/
```

Das ist ein modularer Monolith, keine Microservice-Architektur. Die fachliche Komplexität rechtfertigt Value Objects und Use Cases, aber keine verteilten Deployments.

## 9. Domain-Modell und Ports

### 9.1 Value Objects

- `BranchFamily`
- `TicketKey`
- `TicketNumber`
- `TicketID`
- `BranchSlug`
- `BranchName`
- `SemanticVersion`
- `SupportVersion`
- `CommitType`
- `CommitMessage`
- `PublicationState`
- `TargetBase`

Alle Value Objects werden nur valide erzeugt. Regexe sind Implementierungsbausteine, nicht das gesamte Domain-Modell.

### 9.2 Application Use Cases

- `CreateBranch`
- `ValidateBranch`
- `CreateCommit`
- `ValidateCommit`
- `StartTicketWorkflow`
- `PublishTicketWorkflow`
- `StartHotfixWorkflow`
- `CutReleaseWorkflow`
- `SyncTargetBase`
- `ValidatePrePush`
- `ManageKnownKeys`

### 9.3 Ports

- `GitRepository`: Repository-Zustand lesen und explizite Git-Operationen ausführen
- `KeyPolicy`: Key syntaktisch oder später gegen ein signiertes Bundle prüfen
- `PreferencesStore`: bekannte Keys und UX-Präferenzen speichern
- `Prompt`: interaktive Eingabe und Auswahl
- `Reporter`: Human- oder JSON-Ausgabe

Der anfängliche `SyntaxOnlyKeyPolicy` akzeptiert jeden syntaktisch gültigen Key. Ein späterer `BundleKeyPolicy` darf ihn ohne Änderung der Use Cases ersetzen.

## 10. Kanonische CLI-Oberfläche

Eine Binary stellt getrennte Subcommands für getrennte Use Cases bereit:

```text
git governance branch list
git governance branch create
git governance branch validate
git governance branch sync-base

git governance commit create
git governance commit validate

git governance workflow ticket start
git governance workflow ticket publish
git governance workflow hotfix start
git governance workflow release cut
git governance workflow release backmerge

git governance validate pre-push

git governance config key list
git governance config key add
git governance config key remove
git governance config key set-default

git governance doctor
git governance completion <shell>
```

Es gibt keine getrennten Produkte `mkbranch` und `mkcommit`. Zwei Binaries würden Versionierung, Installation, Konfiguration und Fehlerverträge ohne fachlichen Nutzen duplizieren. Die Subcommands bleiben trotzdem klar getrennt und skriptfähig.

## 11. Workflow-Grenzen

### 11.1 Regulärer Ticket-Start

1. Repository, Remote und sauberer Arbeitsbaum prüfen.
2. `git fetch --prune <remote>`.
3. Ticket und Branch-Familie validieren.
4. Lokale und ausgewählte Remote-Tracking-Branches auf einen bestehenden
   offiziellen regulären Branch mit derselben Ticket-ID prüfen.
5. Offiziellen Branch direkt von `<remote>/develop` erzeugen.
6. Optional einen `scratch/*`-Branch vom lokalen offiziellen Branch erzeugen.
7. Auf dem gewählten Arbeitsbranch enden.

Die interaktive Erklärung muss deutlich machen:

- Scratch ist nur für unsichere Exploration.
- Scratch ist privat und kein PR-Ziel.
- Stabile Arbeit gehört auf den offiziellen Ticket-Branch.
- Scratch-Ergebnisse werden kontrolliert per Squash oder Cherry-Pick übernommen.

### 11.2 Ticket-Veröffentlichung und PR-Vorbereitung

1. Offiziellen Branch und Ticketkonsistenz prüfen.
2. Lokale Governance- und projektspezifische Quality Checks ausführen.
3. `fetch --prune`.
4. Basisfrische prüfen.
5. Nur bei unveröffentlichtem Branch und fehlenden Basis-Commits rebasen.
6. Validierungen erneut ausführen.
7. Ersten Push mit Upstream ausführen oder als Plan ausgeben.
8. Providerneutrale PR-Daten für Ziel `develop` erzeugen.

Ein späterer GitHub-/GitLab-Adapter ist ein Outbound Adapter. Ohne festgelegten Provider darf der Domain-Kern keine `gh`-, GitHub- oder GitLab-Abhängigkeit besitzen.

### 11.3 Hotfix

Ein Hotfix startet von der real betroffenen Linie: `main`, derselben `release/*`-Linie oder derselben `support/*`-Linie. Die Linie ist eine Pflichtangabe und wird nicht aus Bequemlichkeit auf `develop` gesetzt.

### 11.4 Release und Support

`release/*` und `support/*` werden nicht über den normalen Branch-Wizard erzeugt. Der Wizard zeigt sie vollständig an, verweist aber auf die governance-gebundenen Workflow-Kommandos. `main` und `develop` werden erklärt, aber nie als normale Arbeitsbranch-Auswahl angeboten.

## 12. Lefthook: Ergänzung statt Ersatz

Lefthook ist laut eigener Dokumentation ein Git-Hook-Manager: Konfigurationen werden in `.git/hooks` installiert und `lefthook run <hook-name>` führt konfigurierte Jobs aus. Eigene Hooks und interaktive Jobs sind möglich. Das macht Lefthook zu einem guten Runner, aber nicht zu einem Branch-/Commit-/Workflow-Domainprodukt.

Gewichtete Funktionsabdeckung des hier definierten Produkts:

| Fähigkeitsbereich | Gewicht | Lefthook allein | Go-CLI allein | Kombiniertes Ziel |
|---|---:|---:|---:|---:|
| Policy- und Validierungslogik | 25 % | 0 % | 25 % | 25 % |
| Branch-/Commit-Mutationen | 20 % | 0 % | 20 % | 20 % |
| Workflow-Orchestrierung | 20 % | 0 % | 20 % | 20 % |
| Interaktive und maschinelle UX | 10 % | 1 % | 10 % | 10 % |
| Benutzerpräferenzen | 5 % | 0 % | 5 % | 5 % |
| Lokale Hook-Orchestrierung | 10 % | 10 % | 0 % | 10 % |
| CI-/Enforcement-Vertrag | 10 % | 4 % | 6 % | 10 % |
| **Gesamt** | **100 %** | **15 %** | **86 %** | **100 %** |

Die Werte messen native Verantwortung, nicht die Fähigkeit, beliebige Fremdprogramme zu starten. Wenn Lefthook ein externes Go-Programm aufruft, stammt dessen Abdeckung vom Go-Programm.

Zielintegration:

```yaml
commit-msg:
  jobs:
    - run: git-governance commit validate --message-file "{1}" --output human

pre-push:
  jobs:
    - run: git-governance --interactive never validate pre-push --remote "{1}" --output human
      use_stdin: true
```

Keine Regex und keine Git-Workflow-Logik werden in `lefthook.yml` dupliziert.

Quellen:

- [Lefthook-Grundmodell](https://lefthook.dev/)
- [Eigene Lefthook-Hooks](https://lefthook.dev/configuration/Hook/)
- [`lefthook run`](https://lefthook.dev/usage/commands/run/)

## 13. Konfiguration und Key-Speicherung

Bekannte Keys sind UX-Präferenzen, keine Policy Registry. Sie werden über `os.UserConfigDir()` gespeichert:

- Linux: `$XDG_CONFIG_HOME/git-governance/config.json`, sonst `$HOME/.config/git-governance/config.json`
- macOS: `$HOME/Library/Application Support/git-governance/config.json`
- Windows: `%AppData%\git-governance\config.json`

Damit ist `$HOME/.config` nicht auf allen Betriebssystemen der korrekte Standard.

Gespeichert werden:

- bekannte Keys
- optionaler Default-Key
- Accessibility- und Darstellungspräferenzen
- Schema-Version

Ticketnummern werden nicht als globaler Default wiederverwendet. Sie sind arbeitsspezifisch und werden bei Commits aus dem aktuellen Branch abgeleitet. Das verhindert versehentliche Commits für ein altes Ticket.

Quelle: [Go `os.UserConfigDir`](https://pkg.go.dev/os#UserConfigDir)

## 14. Fehlervertrag

Jeder fachliche Fehler besitzt:

- stabilen Code
- Kategorie
- betroffenes Feld
- beobachteten Wert
- verletzte Regel
- erwartetes Format
- gültiges Beispiel
- konkrete Behebung
- kausale technische Fehlermeldung, sofern vorhanden

Beispiel:

```text
BRANCH_SLUG_INVALID
Feld: slug
Wert: add--Export
Regel: Slugs sind kanonisches kebab-case ohne leere Segmente.
Erwartet: [a-z0-9]+(?:-[a-z0-9]+)*
Beispiel: add-export-button
Behebung: Verwende Kleinbuchstaben und einfache Bindestriche.
```

Exitcodes:

- `0`: Erfolg
- `2`: CLI-Nutzung oder fehlende Eingabe
- `3`: Governance-/Validierungsverstoß
- `4`: ungültiger Repository-Zustand
- `5`: Git-Operation fehlgeschlagen
- `6`: Konfiguration oder Policy-Bundle ungültig
- `7`: externer Adapter oder Netzwerk fehlgeschlagen
- `130`: Benutzerabbruch

Im JSON-Modus geht genau ein versioniertes Resultat auf stdout; Diagnosen gehen auf stderr.

## 15. Security und Reliability

- Git wird ausschließlich mit Argumentarrays aufgerufen, nie über Shell-Command-Strings.
- Repository-, Ref- und Pfadwerte werden vor Prozessaufrufen validiert.
- Keine impliziten `git add .`, `push`, Rebase-, Merge- oder Force-Operationen.
- Mutierende Schritte unterstützen `--dry-run` und eine sichtbare Planphase.
- Der Arbeitsbaum muss vor Switch, Rebase und Merge sauber sein.
- Cancellation wird über `context.Context` an Git-Prozesse propagiert.
- Keine Fire-and-forget-Goroutines; Git-Schritte laufen bewusst sequenziell.
- Konfigurationsdateien werden mit plattformgerechter Recovery-Strategie ersetzt
  und mit restriktiven Rechten angelegt.
- Policy-Bundles benötigen später Version, Herkunft, Signatur/Checksumme und Staleness-Regel.
- Hooks sind lokale Frühprüfung; CI und Remote Protection bleiben bindend.

## 16. Build, Release und Installation

Release-Artefakte werden in CI reproduzierbar für die unterstützten OS-/Architekturpaare gebaut, getestet, mit SHA-256 versehen, signiert und zusammen mit SBOM und Provenance veröffentlicht. Package Manager sind der primäre Installationsweg; direkte Archive sind der kontrollierte Fallback.

Nach einem geschützten `release/<semver> -> main`-Merge erzeugt ein
privilegienminimierter GitHub-Actions-Workflow `v<semver>` als annotierten,
unveränderlichen Tag auf genau dem Merge-Commit. Weil ein durch `GITHUB_TOKEN`
erzeugter Tag keinen weiteren Push-Workflow auslöst, startet dieser Workflow
den vorhandenen Artefaktworkflow ausdrücklich per `workflow_dispatch`.

Installationsskripte dürfen nicht ungefragt `.bashrc`, `.zshrc` oder PowerShell-Profile verändern. Details stehen in `docs/operations/INSTALLATION-UND-RELEASE.md`.

## 17. Verifikation

Pflichtgates:

- Domain-Unit- und Table-Driven-Tests für jede gültige und ungültige Grammatikklasse
- Fuzztests für Branch- und Commit-Parser
- Contract Tests für Human- und JSON-Fehler
- Integrationstests gegen temporäre echte Git-Repositories
- Tests für unveröffentlichten und veröffentlichten Branch
- Tests für no-op, Rebase- und Merge-Fälle der Basisfrische
- `go test ./...`
- `go run ./cmd/check-coverage`
- `go test -race ./...`
- `go vet ./...`
- Vulnerability- und Dependency-Scan
- Builds und Smoke Tests auf Windows, macOS und Linux
- Installations- und Upgrade-Tests je Package Manager

Aktueller Verifikationsstatus: Der modulare Go-Kern, die CLI, lokale
Git-Integration, Whitebox-Tests und Fuzz-Smokes werden im Repository
implementiert. Die verbindliche aktuelle Evidenz wird ausschließlich in
`docs/TRACEABILITY.md` geführt. Release-Publikation, native macOS-/Linux-Smokes,
Signierung, Package-Manager-Publikation und Remote-Branch-Protection bleiben
separate, noch nicht lokal nachweisbare Delivery-Gates.

## 18. Konsequenzen

Positiv:

- eine fachliche Wahrheit
- identisches Verhalten auf Windows, macOS und Linux
- gleiche Use Cases für interaktive Nutzung, Automation, Lefthook und CI
- stabile maschinenlesbare Fehler
- spätere Policy-Registry-Integration ohne Umbau der Workflows

Negativ:

- native Artefakte müssen pro OS/Architektur gebaut und getestet werden
- Cobra und Huh erweitern die Supply-Chain-Fläche
- Git bleibt eine erforderliche externe Prozessabhängigkeit
- provider-spezifische PR-Erstellung benötigt einen späteren Adapter

## 19. Verwarfene Optionen

- Parallele `ps1`-/`sh`-Implementierungen: doppelte fachliche Wahrheit
- Lefthook-only: Runner ohne Domain- und Workflow-Fähigkeiten
- Zwei unabhängige Befehle `mkbranch` und `mkcommit`: doppelte Delivery- und Konfigurationsfläche
- TypeScript/Node als Kern: zusätzliche Runtime-Surface ohne fachlichen Nutzen
- Automatisches Editieren von Shell-Profilen: fragil, nicht idempotent und unnötig bei Package-Managern
- Blindes `pull` oder Rebase: mutiert Zustand ohne vorherige Entscheidung
- Interne CLI-Selbstaufrufe: Prozesskopplung statt Application-Service-Komposition
- Ein kontinuierlicher Ticket-bis-PR-Wizard: unpassender langlebiger Workflow über die Entwicklungsphase

## 20. Revisit-Trigger

Die Entscheidung wird neu bewertet, wenn:

- ein Zielbetriebssystem keine Go-Binary zulässt
- harte Echtzeit-, FIPS- oder neue Compliance-Gates entstehen
- ein zentraler Policy-Service mit nachgewiesenem Mehrwert verbindlich wird
- Offline-Fähigkeit entfällt
- providerübergreifende PR-Erstellung zum eigenen Core-Produkt wird
- ein stabiler veröffentlichter CLI-Vertrag eine Aufteilung in mehrere Artefakte erzwingt

## 21. Externe Referenzen

- [Go Release History](https://go.dev/doc/devel/release)
- [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)
- [Semantic Versioning 2.0.0](https://semver.org/)
- [Git Fetch](https://git-scm.com/docs/git-fetch)
- [Git Rebase](https://git-scm.com/docs/git-rebase)
- [Cobra](https://cobra.dev/)
- [Huh](https://github.com/charmbracelet/huh)
- [GoReleaser](https://goreleaser.com/)
