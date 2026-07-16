# Create branches

## Create a regular ticket branch

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
