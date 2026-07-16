# Clean up a scratch branch

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

For automation:

```powershell
git governance --interactive never --output json --yes workflow cleanup `
  --branch scratch/ABC-123-export-exploration
```
