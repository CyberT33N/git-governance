# GitHub: `develop` integration line

## Authority

This document is the **single source of truth** for the GitHub hosting contract
of the shared `develop` line. Product workflow meaning stays in the architecture
and usage docs; local rebase-or-merge rules for *ticket branches* stay in the
policy and synchronization guides. Do not restate those contracts here except
to show how GitHub enforces the integration-line outcome.

| Concern | Authority |
|---|---|
| Workflow topology and PR target `develop` | [ADR-0001 §11](../../architecture/ADR-0001-GO-CLI-ZIELARCHITEKTUR.md), [ticket publish](../../usage/workflows/tickets/publish.md) |
| Local ticket-branch sync (rebase before first push; merge after) | [Policy §6](../../specification/POLICY-UND-VALIDIERUNG.md), [Base synchronization](../../usage/branches/synchronization.md) |
| GitHub merge method and linear history on `develop` | **this document** |

## Architectural basis

Regular ticket workflows treat `develop` as the protected integration line:

1. `workflow ticket start` creates the official branch from `origin/develop`.
2. `workflow ticket publish` prepares a pull request whose default target is
   `develop`.
3. After the first push, an official ticket branch is append-only locally: the
   CLI does not routine-rebase published work onto an advancing base.
4. Remote branch protection remains the binding last instance for what may land
   on shared lines (ADR-0001).

Therefore the **history of `develop` itself** must stay linear at the hosting
boundary. Merge commits into `develop` would create a second integration
topology that the governed ticket workflows do not model. GitHub must accept
only rebase merges onto `develop`.

This hosting rule is complementary to, not a replacement for, the local
publication-state rebase policy on ticket branches.

## Required GitHub contract

`develop` **MUST** be rebase-only managed on GitHub:

1. Pull requests into `develop` **MUST** use **Rebase and merge** only.
2. **Merge commits** into `develop` **MUST** be disabled.
3. **Squash merging** into `develop` **MUST** be disabled for this line so the
   reviewed commit series remains the integration history.
4. Branch protection or a ruleset for `develop` **MUST** require a pull request
   and **MUST** require a linear history.

Direct pushes to `develop` remain forbidden for ordinary contributors; the CLI
treats `develop` as a shared line, not a normal work branch.

## Configure the contract

### Repository pull-request methods

If the repository uses a single global merge-method set, configure:

1. Repository → **Settings** → **General** → **Pull Requests**
2. Enable **Allow rebase merging**
3. Disable **Allow merge commits**
4. Disable **Allow squash merging**

When other shared lines need a different method later, prefer a
branch ruleset scoped to `develop` rather than weakening this contract.

### Branch ruleset for `develop`

1. Repository → **Settings** → **Rules** → **Rulesets**
2. Target branch: `develop`
3. Require a pull request before merging
4. Require linear history
5. Keep the merge method restricted to rebase as above

Equivalent repository flags via GitHub CLI:

```powershell
gh api repos/{owner}/{repo} -X PATCH `
  -f allow_merge_commit=false `
  -f allow_squash_merge=false `
  -f allow_rebase_merge=true
```

Replace `{owner}/{repo}` with this repository's GitHub coordinates.

## Relationship to CLI sync strategies

| Situation | Local CLI behavior | GitHub `develop` landing |
|---|---|---|
| Unpublished official ticket branch, base advanced | conditional rebase onto the target base | not yet applicable |
| Published official ticket branch, base advanced | controlled merge commit on the ticket branch | still lands on `develop` only via rebase merge of the PR |
| `workflow ticket publish` PR intent | provider-neutral target `develop` | rebase-only merge when the PR is accepted |

Operators must not interpret “rebase-only on `develop`” as permission to
force-push or routine-rebase a published official ticket branch. Those remain
policy violations enforced by the CLI and by protection against force pushes.

## Non-goals

- GitLab or other hosts (add sibling directories under `../` when adopted)
- Redefining commit or branch grammar
- Replacing `git-governance` publication workflows
- Documenting GitHub App authentication (see [authentication](../../usage/authentication.md))
