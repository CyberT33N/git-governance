# git-governance

`git-governance` is a native Go CLI for governed Git work on Windows, macOS,
and Linux. It creates and validates branches and commits, guides bounded
ticket workflows, and exposes the same validation core to Lefthook and CI.

The binary is named `git-governance`. When it is on `PATH`, Git also exposes it
as:

```text
git governance ...
```

No Go, Node.js, Python, PowerShell, or shell runtime is required on end-user
machines once a release binary is installed. Git itself is required.

## What the tool owns

- canonical branch, ticket, slug, SemVer, and Conventional Commit validation
- branch creation from the correct remote-tracking base
- commit creation with explicit staging only
- bounded ticket, hotfix, release, support, and backmerge workflows
- first-push base-freshness checks
- stable human and JSON error contracts
- known ticket-key preferences in the operating-system configuration directory
- local Lefthook integration through the same validator commands

The tool does not replace Git hosting, branch protection, CI, a ticket tracker,
or a policy registry.

## Architecture

```text
Cobra / Huh / JSON
        ↓
Application use cases
        ↓
Branch, ticket, and commit domain models
        ↓
Application-owned ports
        ↓
Git CLI, config filesystem, terminal, and reporter adapters
```

The domain never depends on Cobra, Huh, Git process APIs, environment
variables, filesystem paths, or a hosting provider.

## Prerequisites for contributors

- Git 2.53 or newer recommended
- Go 1.26.5 (enforced by the `toolchain go1.26.5` directive)

Check the local environment:

```powershell
go version
git --version
```

Expected Go output begins with:

```text
go version go1.26.5
```

## Build from source

Clone or open the repository, then run the full build gate:

```powershell
go run .\cmd\build
```

On macOS or Linux:

```bash
go run ./cmd/build
```

`cmd/build` verifies root and build-tool module integrity, checks formatting,
runs Staticcheck, typechecks packages and tests, runs unit, contract,
integration, coverage, race, vet, vulnerability, fuzz, and Lefthook checks,
then builds and smoke-tests the native binary. It stops at the first failed
gate and writes `dist\git-governance.exe` on Windows or
`dist/git-governance` on macOS and Linux.

For a local development run without producing a binary:

```powershell
go run .\cmd\git-governance --help
```

To use the Git subcommand form locally, put the built binary in a directory
already on `PATH`:

```powershell
git governance --help
```

Release installers and package-manager manifests are added by the release
pipeline. They are not yet published by this repository.

## Global options

```text
--interactive auto|always|never   default: auto
--output human|json               default: human
--quiet                           suppress successful human output
--color auto|always|never         default: auto
--accessible                      use screen-reader-friendly prompts
--remote <name>                   default: origin
--repo <path>                     default: current directory
--config <path>                   explicit preferences file
--quality-config <path>           repository-local quality gate configuration
--dry-run                         generate a plan without Git mutation
--yes                             confirm a mutation non-interactively
--timeout <duration>              default: 30s
```

`--interactive`, `--output`, and `--quiet` have separate meanings:

- `--interactive never` disables all prompts and requires every mandatory
  value as a flag.
- `--interactive always` requires a real terminal on both standard input and
  standard output; it fails before a workflow starts when no terminal exists.
- `--output json` writes one versioned JSON result to stdout.
- `--quiet` suppresses successful human output only.
- `--color auto` emits ANSI color only to a terminal, `--color always` forces
  it, and `--color never` uses plain human output and line-oriented prompts.
- `--yes` is required for a mutating non-interactive command unless
  `--dry-run` is used.

When an interactive human workflow completes a remote refresh, its success
header starts with:

```text
🟢 Remote references fetched and stale references pruned from <remote> before this operation.
```

The status appears only after a successful `git fetch --prune <remote>` and is
absent from dry runs, `--interactive never`, JSON output, and `--quiet`.
Fetching refreshes configured remote-tracking references; it does not pull or
switch a local branch.

## Interactive input validation

Every interactive field explains its canonical input contract before accepting a
value. If a value is invalid, the command stays at that exact field, prints the
actual value (when safe), the violated rule, the expected format, a valid
example, and the correction. The prompt can be retried without limit; accepted
earlier values are retained and the workflow does not restart.

An error after accepted inputs—for example a repository or Git failure—includes
the input summary used by that command. Inline field diagnostics intentionally
show only the failing field so a correction remains focused. Git failures
separate operation context from the bounded Git diagnostic; the context is not
mislabelled as an input value.

## Project-agnostic quality gates

The CLI does not guess a project's language, package manager, build command,
or linter. Hardcoding `go test`, `npm test`, `mvn test`, or any other stack
command would make a generic Git tool architecturally incorrect.

Instead, a trusted repository can opt in with
`git-governance.quality.json` at its root, or an explicit
`--quality-config <path>`. Commands are executed as an executable plus argument
array; no shell command string is interpreted.

```json
{
  "schemaVersion": 2,
  "defaults": {
    "includeFamilies": [
      "feature", "fix", "docs", "refactor",
      "chore", "test", "perf", "hotfix"
    ]
  },
  "gates": [
    {
      "name": "unit-tests",
      "command": "go",
      "args": ["test", "./..."],
      "timeout": "2m"
    },
    {
      "name": "documentation-links",
      "command": "npm",
      "args": ["run", "docs:check"],
      "includeFamilies": ["docs"],
      "timeout": "2m"
    },
    {
      "name": "stress",
      "command": "./scripts/stress-test",
      "includeFamilies": ["feature", "perf"],
      "timeout": "2m"
    }
  ]
}
```

Each gate uses a repository-relative working directory and a positive Go
duration timeout. Paths that escape the repository, shell control characters,
unknown JSON fields, duplicate gate names, and unbounded configuration are
rejected.

If no configuration exists, the workflow reports `qualityStatus:
unconfigured`; it does not claim that project-specific checks passed. The
configuration is a trust boundary because running project-defined commands can
execute project code. Review it before using it in an unfamiliar repository.

When a valid configuration exists, each gate receives a typed branch-family
scope. A gate without `includeFamilies` or `excludeFamilies` inherits
`defaults`. `includeFamilies` selects only the listed families;
`excludeFamilies` is applied afterward and removes specific families. A
multi-ref push runs every eligible gate once after its per-ref governance
checks pass.

The recommended default includes every official working family:
`feature`, `fix`, `docs`, `refactor`, `chore`, `test`, `perf`, and `hotfix`.
`scratch` is absent from that default because it is private exploration. It is
not a hardcoded exception: a repository can include `scratch` for one small
gate without enabling expensive gates there. This lets a documentation branch
run link checks while skipping a stress test, or a performance branch run a
stress test that other families do not need.

## Command overview

```text
git governance branch list
git governance branch create
git governance branch validate
git governance branch merge-scratch
git governance branch sync-base

git governance commit create
git governance commit validate

git governance workflow ticket start
git governance workflow ticket publish
git governance workflow hotfix start
git governance workflow hotfix publish
git governance workflow hotfix propagate
git governance workflow release cut
git governance workflow release stabilize
git governance workflow release publish-stabilization
git governance workflow release promote
git governance workflow release backmerge
git governance workflow release support
git governance workflow cleanup

git governance validate pre-push

git governance config key list
git governance config key add
git governance config key remove
git governance config key set-default

git governance policy describe
git governance doctor
git governance completion <shell>
```

## Branches

List every supported family and its purpose:

```powershell
git governance branch list
git governance --output json branch list
```

The complete taxonomy is:

| Family | Name pattern | Purpose |
|---|---|---|
| `main` | `main` | published production truth |
| `develop` | `develop` | integration line |
| `release` | `release/<semver>` | frozen release candidate |
| `support` | `support/<major.minor>` | maintained older version line |
| `feature` | `feature/<ticket>-<slug>` | new capability |
| `fix` | `fix/<ticket>-<slug>` | regular defect correction |
| `docs` | `docs/<ticket>-<slug>` | documentation |
| `refactor` | `refactor/<ticket>-<slug>` | internal restructuring |
| `chore` | `chore/<ticket>-<slug>` | maintenance or tooling |
| `test` | `test/<ticket>-<slug>` | test work |
| `perf` | `perf/<ticket>-<slug>` | performance work |
| `hotfix` | `hotfix/<ticket>-<slug>` | defect on an active line |
| `scratch` | `scratch/<ticket>-<slug>` | private exploration |

`feature` is a branch family. `feat` is a commit type and is intentionally not
a valid branch family.

### Create a regular ticket branch

Interactive:

```powershell
git governance branch create
```

Non-interactive:

```powershell
git governance --interactive never --yes branch create `
  --family feature `
  --key ABC `
  --ticket 123 `
  --slug add-export-button
```

The result is:

```text
feature/ABC-123-add-export-button
```

Regular ticket branches always start from `origin/develop`. The tool uses:

```text
git fetch --prune origin
git switch -c feature/ABC-123-add-export-button origin/develop
```

It does not check out and pull the local `develop` branch first.
The selected remote must therefore expose `develop` after the fetch. A
repository adopting this policy must create and protect that integration line
before regular ticket work begins; the CLI now stops with a target-base
diagnostic before attempting `git switch` when it is absent.

Normal ticket work permits exactly one official regular branch per ticket. For
example, once `feature/ABC-123-add-export-button` exists locally or on the
selected remote, a second regular `fix/ABC-123-...` branch is rejected. This
does not prevent explicitly governed release stabilization or hotfix
propagation branches, which have their own active-line lifecycle.

Use `--dry-run` to inspect the plan:

```powershell
git governance --interactive never --dry-run branch create `
  --family feature --key ABC --ticket 123 --slug add-export-button
```

### Create a private scratch branch directly

A scratch branch must start from the local official branch for the same ticket.
The interactive flow asks for that base; non-interactive use supplies it
explicitly:

```powershell
git governance --interactive never --yes branch create `
  --family scratch `
  --key ABC `
  --ticket 123 `
  --slug export-exploration `
  --base feature/ABC-123-add-export-button
```

`origin/feature/...`, a shared line, another scratch branch, and a branch for a
different ticket are rejected. Use `workflow ticket start --scratch` when a new
official branch and its private exploration branch should be created together.

### Squash a scratch branch into its official branch

`branch merge-scratch` transfers the current `scratch/*` branch as exactly one
governed commit to its local official ticket branch. It resolves the target by
ticket ID, not by the slug: `scratch/ABC-123-export-exploration` may therefore
transfer to `feature/ABC-123-add-export-button`, `fix/ABC-123-...`, or another
official family for `ABC-123`. The two descriptions do not need to match.

When exactly one local official branch has the same ticket, no target input is
needed. If no local official branch exists, the command stops with
`SCRATCH_TARGET_BRANCH_MISSING`. If manual work created multiple local
official branches for the ticket, specify the intended target explicitly:

```powershell
git governance branch merge-scratch `
  --target feature/ABC-123-add-export-button `
  --message "feat(ABC-123): add export button"
```

The standard human flow shows the scratch source, resolved target, and squash
commit before requiring confirmation. Non-interactive automation uses the
existing interaction contract rather than a separate `--silent` flag:

```powershell
git governance --interactive never --yes branch merge-scratch `
  --message "feat(ABC-123): add export button"
```

The command switches to the official branch, applies `git merge --squash`, and
creates the supplied Conventional Commit. It never runs `git add .`, pushes,
or deletes the scratch branch. A conflict remains in Git for explicit user
resolution; the CLI does not hide or automatically discard it.

### Validate a branch

```powershell
git governance branch validate
git governance branch validate --branch feature/ABC-123-add-export-button
```

### Synchronize the target base

```powershell
git governance branch sync-base --strategy check
git governance --yes branch sync-base --strategy rebase
git governance --yes branch sync-base --strategy merge `
  --merge-message "chore(ABC-123): merge origin/develop"
```

Policy:

1. Fetch the selected remote.
2. Compare `HEAD` with the real target base.
3. If no base commits are missing, do nothing.
4. An unpublished official branch may rebase only if the base advanced.
5. A published official branch never routine-rebases; a controlled merge is
   required instead.
6. `scratch/*` remains private and is not a pull-request branch.

## Commits

The canonical header is:

```text
<type>(<KEY-NUMBER>)[!]: <subject>
```

Supported types:

```text
build chore ci docs feat fix perf refactor revert style test
```

Examples:

```text
feat(ABC-123): add export button
fix(ABC-123): address review feedback
docs(ABC-123): document export workflow
feat(ABC-123)!: replace the export contract
```

Create a commit:

```powershell
git governance --yes commit create `
  --type feat `
  --subject "add export button" `
  --stage cmd/git-governance/main.go `
  --stage README.md
```

The ticket defaults to the current ticket branch. An explicit ticket must
match that branch.

Add a breaking change:

```powershell
git governance --yes commit create `
  --type feat `
  --subject "replace export contract" `
  --breaking `
  --breaking-description "Clients must read the new resource envelope." `
  --stage internal/domain/commitmsg/message.go
```

Validate a message file, for example from a hook:

```powershell
git governance --interactive never commit validate `
  --message-file .git/COMMIT_EDITMSG
```

The tool never runs `git add .`, `git commit --amend`, or `git push --force`.

## Ticket workflow

Start regular work:

```powershell
git governance workflow ticket start `
  --family feature `
  --key ABC `
  --ticket 123 `
  --slug add-export-button `
  --scratch
```

The workflow:

1. validates the ticket and branch family;
2. fetches `origin`;
3. creates the official branch from `origin/develop`;
4. optionally creates `scratch/ABC-123-add-export-button-exploration` from
   the local official branch;
5. ends on the selected active branch.

Scratch is only for uncertain exploration. Never open a pull request from it.
Move stable work to the official branch through a controlled squash or
cherry-pick.

Publish after development:

```powershell
git governance --yes workflow ticket publish --push
```

When invoked from an official ticket branch, publication follows the normal
flow. When invoked from a `scratch/*` branch, the workflow first resolves the
same local official ticket branch, shows both branch names and the supplied
squash commit, and asks for confirmation. After confirmation it reuses the
same `branch merge-scratch` application component before validating and
optionally pushing the official branch:

```powershell
git governance workflow ticket publish `
  --message "feat(ABC-123): add export button" `
  --push
```

For non-interactive execution, use `--interactive never --yes` and provide
the commit message. Use `--target <official-branch>` only when a manually
created repository has more than one local official branch for the ticket.

The workflow validates the commit series, runs configured quality gates,
checks base freshness, conditionally rebases only an unpublished branch, then
revalidates branch policy, the full commit series, and quality gates before it
can push. It emits a provider-neutral pull-request intent targeting `develop`
and deliberately ends at that pull-request boundary.

## Hotfix, release, and support workflows

Hotfix:

```powershell
git governance --yes workflow hotfix start `
  --key ABC `
  --ticket 999 `
  --slug payment-timeout `
  --affected-line main
```

The affected line must be `main`, `release/<semver>`, or
`support/<major.minor>`. A hotfix never starts from `develop` by default.
The selected remote base is stored only in local Git metadata so later
publish, sync, and pre-push validation use the same affected line.

Publish the hotfix to the same affected line:

```powershell
git governance --yes workflow hotfix publish `
  --affected-line main `
  --push
```

Forward-port or backport one reviewed hotfix commit. The command creates a
short-lived `fix/*` branch from the requested target line, applies
`git cherry-pick -x`, validates it, and emits a pull-request intent:

```powershell
git governance --yes workflow hotfix propagate `
  --target-line develop `
  --commit 0123456789abcdef0123456789abcdef01234567 `
  --push
```

Release cut:

```powershell
git governance --yes workflow release cut --version 2.8.0
```

The CLI validates the requested `release/2.8.0` line and emits an intent for
the protected `create-protected-line.yml` GitHub Actions workflow. It does not
create, switch to, or push a local `release/*` branch. An authorized
release-environment workflow creates the remote line from `origin/develop`;
developers fetch it before starting controlled stabilization work.

Only release-blocking fixes, final documentation, and release preparation are
allowed after a cut. Create the corresponding short-lived stabilization branch:

```powershell
git governance --yes workflow release stabilize `
  --release release/2.8.0 `
  --kind blocker `
  --key ABC `
  --ticket 999 `
  --slug release-blocker-timeout
```

Publish its pull-request intent back to the frozen release line:

```powershell
git governance --yes workflow release publish-stabilization `
  --release release/2.8.0 `
  --push
```

After approval, prepare the release promotion and later its backmerge:

```powershell
git governance workflow release promote --release release/2.8.0
git governance workflow release backmerge --release release/2.8.0
```

After the protected `release/<semver> -> main` pull request merges, GitHub
Actions creates the immutable annotated `v<semver>` tag at that exact merge
commit. The tag workflow then dispatches the artifact workflow for that tag.
The local CLI never tags or pushes `main`.

Support line:

```powershell
git governance --yes workflow release support --version 2.8
```

The command requires a matching `v2.8.<patch>` release tag on `origin/main`
and emits the same protected-line workflow intent. The privileged workflow
creates the remote support line from the tagged `origin/main` revision;
support lines cannot be created from an untagged integration state.

Clean up a private scratch branch locally:

```powershell
git governance --yes workflow cleanup `
  --branch scratch/ABC-123-export-exploration
```

The CLI never deletes a remote branch or an official working branch. GitHub or
GitLab deletes merged ticket- and hotfix-branch remotes through its configured
merge-request policy. CI or hosting automation retains and later deletes a
release branch only after both promotion to `main` and the backmerge to
`develop` are complete.

`workflow cleanup` is intentionally limited to local `scratch/*` branches.
The CLI does not claim to prove merge, propagation, or hosting lifecycle
completion; those are GitHub/GitLab/CI responsibilities. `main`, `develop`,
`release/*`, `support/*`, and all official ticket branches are never local
cleanup targets.

## Ticket-key preferences

Keys are convenience preferences, not organizational authorization. v1 accepts
every syntactically valid key:

```regex
^[A-Z][A-Z0-9]*$
```

Manage keys:

```powershell
git governance config key add --key PLATFORM2
git governance config key set-default --key PLATFORM2
git governance config key list
git governance config key remove --key PLATFORM2
```

Configuration location uses Go `os.UserConfigDir()`:

| Platform | Default application directory |
|---|---|
| Linux | `$XDG_CONFIG_HOME/git-governance` or `$HOME/.config/git-governance` |
| macOS | `$HOME/Library/Application Support/git-governance` |
| Windows | `%AppData%\git-governance` |

The file is versioned JSON and uses a platform-aware recoverable replacement
strategy. Unix replacement uses same-directory rename semantics; Windows keeps
and restores a temporary `.bak` recovery copy if an interrupted replacement
leaves the target absent. It never stores secrets or a global default ticket
number.

## Lefthook

Install Lefthook through your approved package channel, make
`git-governance` available on `PATH`, then run:

```powershell
lefthook install
lefthook validate
```

The repository `lefthook.yml` is intentionally thin:

- `commit-msg` runs `git-governance commit validate --message-file "{1}"`;
- `pre-push` runs `git-governance validate pre-push` and forwards Git stdin.

No branch regex, ticket policy, network call, rebase, merge, or direct Git
mutation is duplicated in Lefthook configuration.

The `pre-push` validator parses every update supplied by Git, including
`HEAD:main`, multiple ref updates, deletion requests, and non-fast-forward
updates. Shared-line mutations and official-branch force pushes are blocked
against the actual remote target rather than the checked-out branch.

## Diagnostics, policy, and completions

```powershell
git governance doctor
git governance --output json policy describe
git governance completion powershell > git-governance.ps1
```

Generate scripts for `bash`, `zsh`, `fish`, or `powershell`.

`doctor` is read-only. It reports the Git version, repository and history
state, selected remote availability, active rebase/merge/cherry-pick state,
Lefthook binary/configuration status, local policy mode, and user
configuration status.

## Errors and exit codes

All errors have a stable code, field, non-sensitive actual value, rule, valid
example, and remediation. Sensitive values are redacted in both human and JSON
output. In JSON mode, errors have this shape:

```json
{
  "schemaVersion": 1,
  "ok": false,
  "error": {
    "code": "COMMIT_TICKET_MISMATCH",
    "category": "governance",
    "field": "ticket"
  }
}
```

Exit codes:

| Code | Meaning |
|---:|---|
| 0 | success |
| 1 | internal failure |
| 2 | CLI usage or missing input |
| 3 | governance or policy violation |
| 4 | repository state failure |
| 5 | Git operation failure |
| 6 | configuration failure |
| 7 | external adapter failure |
| 130 | cancellation |

## Verification

```powershell
go test ./...
go run .\cmd\check-coverage
CGO_ENABLED=1 go test -race ./...
go vet ./...
go test ./internal/integration -count=1
go install golang.org/x/vuln/cmd/govulncheck@v1.5.0
govulncheck ./...
```

On Windows, the race detector needs a working C compiler and `CGO_ENABLED=1`.
If the local host cannot provide that compiler, use the Ubuntu CI gate rather
than claiming a skipped race run passed.

Fuzz smoke tests:

```powershell
go test ./internal/domain/ticket -run=^$ -fuzz=FuzzParseTicketValues -fuzztime=2s -parallel=1
go test ./internal/domain/branch -run=^$ -fuzz=FuzzParseBranchValues -fuzztime=2s -parallel=1
go test ./internal/domain/commitmsg -run=^$ -fuzz=FuzzParseCommitMessage -fuzztime=2s -parallel=1
go test ./internal/adapters/configfs -run=^$ -fuzz=FuzzDecodePreferences -fuzztime=2s -parallel=1
go test ./internal/adapters/quality -run=^$ -fuzz=FuzzDecodeQualityConfiguration -fuzztime=2s -parallel=1
```

## Further documentation

- [Architecture Decision Record](docs/architecture/ADR-0001-GO-CLI-ZIELARCHITEKTUR.md)
- [Policy and validation contract](docs/specification/POLICY-UND-VALIDIERUNG.md)
- [CLI contract](docs/specification/CLI-VERTRAG.md)
- [Installation and release design](docs/operations/INSTALLATION-UND-RELEASE.md)
- [Product acceptance matrix](docs/TRACEABILITY.md)
- [Contributing](CONTRIBUTING.md)
- [Package-manager publication templates](packaging/README.md)
