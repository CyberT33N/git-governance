# Hotfix workflows

## Start

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

Non-interactive start:

```powershell
git governance --interactive never --output json --yes workflow hotfix start `
  --key ABC `
  --ticket 999 `
  --slug payment-timeout `
  --affected-line main
```

## Publish

Publish the hotfix to the same affected line:

```powershell
git governance --yes workflow hotfix publish `
  --affected-line main `
  --push
```

For automation:

```powershell
git governance --interactive never --output json --yes workflow hotfix publish `
  --affected-line main `
  --push
```

Add `--pull-request-provider github --create-pull-request` to create the
explicit provider-backed pull request after the push. The target remains the
specified affected line; it is never silently redirected to `develop`.

After resolving and staging a paused rebase, continue the same publication:

```powershell
git governance --interactive never --output json --yes workflow hotfix publish `
  --affected-line main `
  --resume `
  --push
```
