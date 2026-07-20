# Release reconciliation

Release reconciliation is the mandatory final assessment of a delivered
`release/<semver>` line against `develop`. It is not an unconditional request
to create an empty pull request.

## Preconditions

Before reconciliation, all of these facts must be authoritative:

1. The `release/<semver> -> main` pull request merged.
2. The immutable `v<semver>` tag points exactly to that promotion merge commit.
3. The configured release artifact workflow completed and published the
   non-draft GitHub Release, or the repository explicitly marks that delivery
   component as not applicable.

The order is causal, not time-based. A release branch is not eligible for
backmerge merely because a main pull request was opened or because a tag name
exists.

## Conditional result

Run the governed reconciliation:

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release backmerge `
  --release release/2.8.0 `
  --create-pull-request
```

The GitHub lifecycle adapter verifies the promotion, tag, and published release
before comparing `release/2.8.0` with `develop`.

| Result | Meaning | Required action |
|---|---|---|
| `required` | A release-only effective delta remains outside `develop`. | Create or use the `release/2.8.0 -> develop` PR and merge it with a merge commit. |
| `not-required` | No effective delta remains. For example, a stabilization change was already independently carried to `develop`. | Do not create an empty PR; retain the returned evidence in the release audit. |

A commit-count difference alone is not final authority. The provider comparison
also checks whether an effective content delta remains.

## Cleanup

The release line remains protected until one reconciliation outcome is proven:

- a required backmerge PR merged successfully; or
- a `not-required` result was recorded with the promotion, tag, release, and
  comparison evidence.

The local CLI never deletes an official release branch. GitHub or controlled CI
performs that lifecycle action after the release record is complete.

## End-to-end verification after a feature release

Run this sequence only after the change under review is merged into `develop`
and the release owner approves a concrete SemVer version:

1. Run `workflow release cut --dispatch` and confirm the returned workflow URL
   succeeded and `origin/release/<semver>` exists.
2. Complete only permitted stabilization work, if any, through its own PRs to
   the frozen release line.
3. Create and merge the reviewed `release/<semver> -> main` promotion PR.
4. Confirm the immutable tag, successful artifact workflow, and published
   GitHub Release.
5. Run `workflow release backmerge --create-pull-request`.
6. Verify either the required `release/<semver> -> develop` PR or the
   `not-required` evidence.
7. Let controlled hosting automation clean the release branch only after that
   outcome.

Do not substitute a local Git push, a manually invented tag, or a no-op pull
request for any of these provider-verified lifecycle steps.
