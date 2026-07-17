# GitHub branch rulesets (importable JSON)

These JSON files are the executable, repository-specific source for GitHub
branch protection. They match the GitHub repository ruleset body used by:

- UI: **Settings → Rules → Rulesets → New ruleset → Import a ruleset**
- API: `POST /repos/{owner}/{repo}/rulesets`

Schema authority:

- [REST API endpoints for rules](https://docs.github.com/en/rest/repos/rules)
- [Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)
- [Managing rulesets for a repository](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository)

The rulesets enforce remote behavior; they do not redefine branch taxonomy,
ticket grammar, or local workflow policy.

## Files

| File | Targets | Intent |
|---|---|---|
| [`01-ticket-working-branches.json`](01-ticket-working-branches.json) | all official ticket and `hotfix/*` working branches | light protection: block force push; allow direct commits and deletion after merge |
| [`02-develop.json`](02-develop.json) | `develop` | strong shared line: PR only, no force push, no deletion, CI/review gates, explicitly permits all three repository-approved PR merge methods |
| [`03-main.json`](03-main.json) | `main` | maximal shared line: PR only, no force push, no deletion, CI/review gates, merge commits only |
| [`04-release.json`](04-release.json) | `release/*` | very strong release line: PR only, no force push, no deletion, CI/review gates, merge commits only |
| [`05-support.json`](05-support.json) | `support/*` | very strong long-lived maintenance line: PR only, no force push, no deletion, CI/review gates, merge commits only |

## Branch-family coverage

| Branch family | Enforcement location | Reason |
|---|---|---|
| `main` | `03-main.json` | Protected production truth; PR-only and not rewriteable |
| `develop` | `02-develop.json` | Protected integration line; PR-only and not rewriteable |
| `release/*` | `04-release.json` | Frozen shared release line; PR-only and not rewriteable |
| `support/*` | `05-support.json` | Long-lived shared maintenance line; PR-only and not rewriteable |
| `feature/*`, `fix/*`, `docs/*`, `refactor/*`, `chore/*`, `test/*`, `perf/*` | `01-ticket-working-branches.json` | Official working branches: direct commits allowed, force pushes prohibited after publication |
| `hotfix/*` | `01-ticket-working-branches.json` plus its protected target line | Official working branch: direct commits allowed, force pushes prohibited; the PR lands on `main`, `release/*`, or `support/*` |
| `scratch/*` | Explicitly no GitHub ruleset | Private, short-lived exploration permits local rebase/amend/force-push behavior and is never a PR source |
| `staging` | Not applicable | The rules define staging as an artifact environment, not a branch |

The absence of a `scratch/*` JSON file is intentional policy coverage, not a
gap. A no-op GitHub ruleset would add no enforcement; a restrictive one would
contradict the private-exploration contract.

## Hotfix, Scratch, and Staging

### Hotfix

`hotfix/*` is a short-lived official working branch for an incident on the
line that actually carries the defect. It is already covered by
[`01-ticket-working-branches.json`](01-ticket-working-branches.json), which
blocks force pushes while preserving direct implementation commits.

There is deliberately no duplicate hotfix-only Ruleset: it would enforce the
same `non_fast_forward` rule twice. The pull request is instead protected by
the Ruleset for its actual destination:

- `hotfix/*` → `main`: [`03-main.json`](03-main.json)
- `hotfix/*` → `release/*`: [`04-release.json`](04-release.json)
- `hotfix/*` → `support/*`: [`05-support.json`](05-support.json)

This preserves the actual incident lineage and avoids treating every hotfix as
if it belonged to `main`.

### Scratch

`scratch/*` is private, short-lived exploration. It is neither an official
pull-request source nor a shared integration line. Its purpose is to allow a
developer to experiment safely before stable work is transferred to an
official ticket branch.

No GitHub Ruleset is created for `scratch/*`: its intended private workflow
permits local rebase, amend, and force-push behavior. A Ruleset that blocked
those operations would defeat the reason the branch exists.

### Staging

`staging` is not a branch and therefore has no Ruleset. It is an environment
fed from release-candidate artifacts built from `release/*`. Treating it as a
permanent branch would introduce an unsupported branch-per-environment flow
and obscure the release artifact that was actually deployed.

## Enforcement boundary

Rulesets are the final remote enforcement layer, not a replacement for the
full governance system. GitHub cannot encode every repository governance
constraint faithfully in static branch rulesets:

- ticket-scoped branch grammar and the equality between branch ticket and
  Conventional Commit scope;
- the mandatory, conditional rebase before an official branch's first push;
- the prohibition of feature work on a frozen `release/*` line;
- the actual affected line, explicit propagation, and `cherry-pick -x`
  provenance of a hotfix;
- a merge-method choice conditional on the PR source branch or on whether a
  ticket history contains internal noise;
- deletion only after release promotion **and** backmerge.

Those controls remain in the `git-governance` domain/application workflow,
Lefthook, CI, and human release governance. Treating GitHub Rulesets as the
only architecture would both leave these constraints unenforced and duplicate
the canonical policy in a less expressive mechanism.

## Merge strategy mapping

GitHub rulesets target the **destination branch**, not the PR source. They
cannot enforce different merge methods for different source families targeting
the same line.

| Destination | Canonical workflow result | Ruleset behavior |
|---|---|---|
| `develop` | Regular ticket PRs may use rebase merge, normal merge, or a selective squash; the release backmerge must remain possible | Explicitly allows `merge`, `rebase`, and `squash` |
| `main` | Release promotion and hotfix lineage prefer a merge commit | `merge` only |
| `release/*` | Controlled stabilization and hotfix lineage prefer a merge commit | `merge` only |
| `support/*` | Controlled maintenance and hotfix lineage prefer a merge commit | `merge` only |

`required_linear_history` is deliberately absent: it would prohibit the merge
commits explicitly preferred for release and hotfix lineage. A GitHub **rebase
merge** is not the same operation as rebasing the shared target branch; the
reference workflow permits it for a regular ticket PR into `develop`, but does
not make it a global or universal requirement. GitHub rulesets cannot express
the remaining conditional decision (“squash only for internal noise”) because
they cannot select a merge method by PR source branch or commit quality.

## Bypass, review, and lifecycle decisions

`bypass_actors` is intentionally empty. GitHub omits bypass lists when
exporting a ruleset, and the repository cannot safely invent team or app IDs.
Do not grant ordinary developers bypass.

`release/*` is protected from premature deletion. Before a completed release
line can be deleted after both its promotion and backmerge, configure an
explicit, audited release-maintainer or release-automation bypass through the
GitHub UI or API. The same identity decision is needed if release or support
branch creation must be restricted to a release workflow. GitHub bypass is
ruleset-wide, so use a dedicated, least-privileged actor rather than an
unrestricted administrator bypass.

The rulesets require one approval and resolved review threads, which implement
the governance recommendations. `require_code_owner_review` is `false` because
this repository currently has no `CODEOWNERS` file; set it to `true` only after
that ownership contract exists. Signed commits are likewise a recommended,
separate decision and are not silently imposed here.

## Import (UI)

1. Open the repository on GitHub.
2. **Settings → Rules → Rulesets**.
3. **New ruleset → Import a ruleset**.
4. Select one JSON file from this directory.
5. Review targets and rules, then **Create**.
6. Repeat for each file in numeric order.

## Import (API)

```powershell
gh api --method POST repos/{owner}/{repo}/rulesets --input docs/platforms/github/rulesets/01-ticket-working-branches.json
gh api --method POST repos/{owner}/{repo}/rulesets --input docs/platforms/github/rulesets/02-develop.json
gh api --method POST repos/{owner}/{repo}/rulesets --input docs/platforms/github/rulesets/03-main.json
gh api --method POST repos/{owner}/{repo}/rulesets --input docs/platforms/github/rulesets/04-release.json
gh api --method POST repos/{owner}/{repo}/rulesets --input docs/platforms/github/rulesets/05-support.json
```

## Status checks

Shared-line rulesets require the CI job names from `.github/workflows/ci.yml`:

- `Quality gates (linux-amd64)`
- `Quality gates (macos-arm64)`
- `Quality gates (windows-amd64)`

If those check names change, update the `required_status_checks` arrays before
import, or adjust them in the GitHub UI after import.

## Required repository merge settings

GitHub's repository settings expose merge-method **capabilities** globally;
Rulesets then restrict their use by destination branch. This repository
**MUST** enable all three settings under **Settings → General → Pull Requests**
so that no supported workflow is blocked:

| Repository setting | Why it must be enabled | Where Rulesets permit it |
|---|---|---|
| **Allow merge commits** | Preserves visible release and hotfix lineage. | `main`, `release/*`, and `support/*` allow only `merge`; `develop` also permits it for a normal merge or release backmerge. |
| **Allow rebase merging** | Preserves a deliberately reviewed regular ticket commit series when it lands on the integration line. | `develop` only. |
| **Allow squash merging** | Supports the exceptional case in which a ticket PR contains internal noise that should become one reviewable integration commit. | `develop` only. |

The global settings do not grant every method to every branch. The effective
method is the intersection of the repository capability and the Ruleset for
the pull request's destination branch:

```text
globally enabled merge methods ∩ destination-branch allowed_merge_methods
```

`02-develop.json` explicitly allows all three methods. That does **not** make
rebase the global default and does not authorize a rebase of `develop` itself.
`03-main.json`, `04-release.json`, and `05-support.json` constrain their
targets to merge commits, preserving release and hotfix lineage despite the
repository-wide capability settings.

No ruleset-only configuration can enforce the source-dependent merge decision
for `develop`; retain the `git-governance` workflow, review policy, and CI as
the authoritative controls for that remaining semantic rule.
