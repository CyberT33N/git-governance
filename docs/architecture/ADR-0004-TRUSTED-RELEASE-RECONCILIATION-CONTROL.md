# ADR-0004: Trusted Release Reconciliation Control

- Status: Proposed
- Datum: 2026-08-01
- Geltungsbereich: Ausführung einer strikten
  `release/<semver> -> develop`-Reconciliation
- Entscheider: Release-Governance

## Kontext

Eine ausgelieferte `release/<semver>`-Ref bleibt nach Promotion, Tag und
Delivery unverändert. Wenn `develop` seit der Promotion neue Commits enthält
und eine aktuelle Pull-Request-Basis erzwingt, wird eine ticketgebundene
Preparation-Branch aus der Release-Ref benötigt.

Der Preparation-Branch enthält absichtlich nicht automatisch die neueste
CLI-Implementierung. Ein Aufruf von `go run ./cmd/git-governance` aus diesem
Worktree würde daher nicht zuverlässig den kontrollierten
Reconciliation-Workflow enthalten. Ein manueller Merge wäre keine zulässige
Alternative.

## Entscheidung

Die geschützte Release-Control-Workflow-Datei auf `main` baut vor jedem
Reconciliation-Branch-Wechsel einen unveränderlichen CLI-Binary aus dem
vertrauenswürdigen Main-Control-Plane-Commit.

```text
trusted main control-plane source
  -> build immutable release-control binary
  -> create release-derived preparation branch
  -> execute binary against that branch
  -> merge current develop only into preparation branch
  -> quality gates and reviewed PR to develop
```

Die Workflow-Operation `reconciliation-align` benötigt Release-Linie,
Ticket-Key, Ticket-Nummer und Slug. Sie erhält eine kurzlebige
Broker-Workload-Identität, konfiguriert den Git-Transport nur im ephemeren
Runner mit einem maskierten Installation-Token und entfernt diese Konfiguration
am Ende des Jobs.

Der Binary führt erst `workflow release stabilize --kind release-prep` und
anschließend `workflow release align-reconciliation-base` aus. Der Broker
bleibt ausschließlich Token-Aussteller; er erzeugt weder Branches noch Pull
Requests.

## Invarianten

- `release/<semver>` wird nie durch den Controller aktualisiert, rebased oder
  direkt gepusht.
- Der Controller akzeptiert nur einen `main`-Workflow-Dispatch im geschützten
  Release-Environment.
- OIDC-Token und Installation-Token werden nicht ausgegeben, persistiert oder
  als Repository-Secret gespeichert.
- Der Git-Transport-Header ist nur im lokalen Runner-Konfigurationsbereich
  vorhanden und wird vor Job-Ende entfernt.
- Der Preparation-Branch trägt den Ticketbezug, startet von der Release-Ref und
  ist der einzige Merge-Ort für den aktuellen Develop-Stand.
- Der resultierende Pull Request zielt auf `develop` und verwendet einen Merge
  Commit.

## Konsequenzen

- Ein nach `develop` gemergter Workflow kann erst nach seiner governeden
  Main-Control-Plane-Promotion privilegiert für eine ausgelieferte Release-Linie
  verwendet werden.
- Dry-Run-Aufrufe bleiben strikt read-only und dürfen weder Provider-Publish
  noch Pull-Request-Erstellung auslösen.
- Die Reconciliation bleibt nachvollziehbar, ohne die veröffentlichte
  Release-Lineage zu verändern.
