# Product Acceptance Matrix

This matrix is the repository-local source of truth for delivery status. It
does not rely on any external governance repository or unpublished rule set.

## Status legend

- `IMPLEMENTED`: source code exists.
- `VERIFIED`: automated tests or an actual local execution succeeded.
- `IN_PROGRESS`: a confirmed gap is actively being remediated.
- `PENDING`: intentionally planned but not yet delivered.
- `BLOCKED`: cannot be verified because an external prerequisite is absent.

## Verified baseline

| Item | Status | Evidence |
|---|---|---|
| Local repository | VERIFIED | `main` and `origin/main` are initialized; the audit baseline was commit `1da3f72` (`added unit tests`) |
| Go toolchain | VERIFIED | Go 1.26.5, Windows amd64 |
| Git client | VERIFIED | Git 2.53.0.windows.2 |
| Legacy scripts and copied hooks | VERIFIED | not present in this repository |
| Go module | VERIFIED | `github.com/CyberT33N/git-governance`, language Go 1.26 and pinned toolchain Go 1.26.5 |

## Core domain and validation

| Capability | Status | Verification |
|---|---|---|
| Typed errors, remediation, exit codes | VERIFIED | whitebox tests |
| Ticket key, number, and ID grammar | VERIFIED | table tests and fuzzing |
| Syntax-only key policy | VERIFIED | unit tests |
| All 13 branch families | VERIFIED | parser and catalog tests |
| Slug, release SemVer, and support version parsing | VERIFIED | table tests and fuzzing |
| Publication-state and rewrite policy | VERIFIED | application tests |
| Conventional Commit parser | VERIFIED | header, body, footer, breaking, revert, and fuzz tests |
| Ticket-to-branch commit consistency | VERIFIED | application tests |
| JSON problem contract | VERIFIED | adapter and CLI tests |
| Human problem contract includes a safe actual value | IMPLEMENTED | non-sensitive actual values are rendered; sensitive values remain redacted |

## Git behavior

| Capability | Status | Verification |
|---|---|---|
| Argument-array Git process execution | VERIFIED | whitebox process-contract tests |
| Bounded stdout and stderr capture | VERIFIED | adapter tests |
| Context and timeout propagation | VERIFIED | adapter tests |
| Branch creation from remote target bases | VERIFIED | real local Git integration test |
| One official regular branch per ticket | VERIFIED | local/remote branch discovery, whitebox test, and real-Git regression test |
| Explicit staging only | VERIFIED | application and Git adapter tests |
| Commit creation through stdin | VERIFIED | real local Git integration test |
| First-push publication detection | VERIFIED | real local Git integration test |
| Base delta, merge, and rebase paths | VERIFIED | real local Git integration test |
| No automatic amend or force push | VERIFIED | absent from public command tree and application APIs |

## User-facing commands

| Command area | Status | Notes |
|---|---|---|
| `branch list`, `validate`, `create`, `sync-base` | IMPLEMENTED | CLI contract tests cover help, JSON, flags, and dry-run behavior |
| `commit create`, `validate` | IMPLEMENTED | explicit staging and ticket consistency are enforced |
| `workflow ticket start` | IMPLEMENTED | optional scratch branch and provider-neutral PR intent |
| `workflow ticket publish` | IMPLEMENTED | regular ticket flow reruns branch, commit-series, and quality checks after a mutation |
| `workflow hotfix start` | IMPLEMENTED | affected-line selection is mandatory |
| hotfix publish and propagation | IMPLEMENTED | affected-line publish plus `cherry-pick -x` forward/backport workflow |
| `workflow release cut`, `stabilize`, `promote`, `backmerge`, `support`, `cleanup` | IMPLEMENTED | stabilization constraints, release-to-main intent, cleanup, and support-tag provenance are enforced |
| `validate pre-push` | IMPLEMENTED | parses every Git stdin ref update and validates the actual remote target |
| `config key` | IMPLEMENTED | OS configuration directory, atomic JSON storage |
| `policy describe`, `completion`, `version` | IMPLEMENTED | policy and environment inspection are read-only |
| `doctor` | IMPLEMENTED | Git version, remote, Lefthook, policy, configuration, and in-progress-operation checks are read-only |
| Interactive Huh forms and accessible prompts | IMPLEMENTED | tested with accessible form input |
| Direct `git governance` invocation | IMPLEMENTED | available when `git-governance` is on `PATH` |

## Workflow policy

| Rule | Status | Behavior |
|---|---|---|
| Regular work starts from `origin/develop` | VERIFIED | direct remote base, no local `develop` checkout/pull required |
| Hotfix starts from actual affected line | VERIFIED | only `main`, `release/*`, or `support/*` accepted |
| Hotfix PR targets actual affected line | IMPLEMENTED | hotfix publish requires and uses the affected main/release/support line |
| Specialized workflow base metadata | VERIFIED | local Git metadata records hotfix, stabilization, and propagation bases for later sync and pre-push validation |
| First push checks basis freshness | VERIFIED | push is blocked when an unpublished branch misses base commits |
| Unpublished branch rebase | VERIFIED | only after a real base delta |
| Published branch synchronization | VERIFIED | recommends or performs explicit merge, never routine rebase |
| Scratch branch | VERIFIED | private local branch from the same-ticket official local branch; remote-tracking bases are rejected |
| Release stabilization and completion | IMPLEMENTED | constrained stabilization, promotion intent, backmerge, cleanup, and support-tag provenance are present |

## Testing and quality

| Gate | Status | Latest local result |
|---|---|---|
| `go test ./...` | VERIFIED | passed after final remediation |
| `go vet ./...` | VERIFIED | passed after final remediation |
| Domain whitebox coverage | VERIFIED | 83.8–100 % by domain package |
| Git adapter whitebox coverage | VERIFIED | 78.5 % |
| Preferences whitebox coverage | VERIFIED | 80.9 % |
| Quality adapter whitebox coverage | VERIFIED | 78.0 % |
| Workflow whitebox coverage | VERIFIED | 73.3 % |
| Bootstrap CLI whitebox coverage | VERIFIED | 60.9 % |
| Terminal adapter whitebox coverage | VERIFIED | 78.8 % |
| Real Git integration | VERIFIED | passed against temporary local bare remotes |
| Bounded fuzzing | VERIFIED | ticket, branch, commit, and configuration targets passed |
| Race detection | VERIFIED | `CGO_ENABLED=1 go test -race ./...` passed locally with GCC 16.1.0 |
| Vulnerability scan | VERIFIED | `govulncheck` v1.5.0 reported no vulnerabilities |
| Windows amd64 native smoke | VERIFIED | version, policy, branch catalog, and doctor commands passed |
| Windows/macOS/Linux cross-builds | VERIFIED | all six promised OS/architecture binaries compiled with `CGO_ENABLED=0` |
| Native ARM64 smoke tests | IMPLEMENTED | CI matrix contains Ubuntu ARM64, Windows ARM64, and macOS ARM64 runners; remote execution requires the first push |
| macOS/Linux native smoke tests | BLOCKED | configured in CI but not executable on the local Windows host |

## Delivery and operations

| Capability | Status | Notes |
|---|---|---|
| Lefthook configuration | IMPLEMENTED | thin `commit-msg` and `pre-push` runners; no duplicated regex |
| Local Lefthook validation | VERIFIED | Lefthook v2.1.10 returned `All good` |
| Reproducible release configuration | VERIFIED | GoReleaser v2.16.0 validated `.goreleaser.yaml` locally |
| GitHub Actions CI | IMPLEMENTED | immutable action commits, pinned tool versions, race, fuzz, vulnerability, Lefthook, native-smoke, and release-config gates are configured |
| GitHub release artifacts | IMPLEMENTED | tag/manual-tag validation, checksums, SBOM, Cosign, provenance attestation, and Linux package formats are configured |
| CI-owned release tag lifecycle | IMPLEMENTED | merged same-repository `release/<semver> -> main` creates an immutable annotated tag and dispatches the artifact workflow |
| Package-manager manifest templates | IMPLEMENTED | Homebrew, Scoop, and WinGet templates are version/checksum-driven under `packaging/` |
| Package-manager publication | BLOCKED | maintainer-controlled tap, bucket, WinGet submission, and publisher identities are external prerequisites |
| Platform-native signing and notarization | BLOCKED | Authenticode and Apple credentials are external publisher prerequisites; checksum Cosign signing remains configured |

## Confirmed remediation work

| Gap | Target remediation | Status |
|---|---|---|
| Pre-push stdin discarded | parse and validate every outgoing update, including deletes and non-fast-forward updates | IMPLEMENTED |
| Hotfix publish target | retain the actual affected line and route the first PR to that line | IMPLEMENTED |
| Post-rebase verification | rerun branch, commit-series, policy, and configured quality validation | IMPLEMENTED |
| Release lifecycle | add stabilization, release-to-main intent, controlled propagation, and cleanup | IMPLEMENTED |
| Direct scratch selection | require/select an official ticket-branch base before creation | IMPLEMENTED |
| Application-level scratch base guard | reject remote-tracking scratch bases even for programmatic callers | IMPLEMENTED |
| Regular ticket exclusivity | reject a second official regular branch for one ticket after fetch | IMPLEMENTED |
| Project-agnostic quality gates | explicit repository-local command-array configuration; absent config reports `unconfigured` instead of pass | IMPLEMENTED |

## Explicit non-goals in v1

- No live ticket-registry lookup.
- No provider-specific pull-request API without an explicit adapter.
- No automatic self-update.
- No direct mutation of protected shared lines.
- No automatic shell-profile editing.
- No compiler or Go SDK requirement on end-user systems.
