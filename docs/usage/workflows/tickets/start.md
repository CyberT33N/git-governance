# Start ticket work

Interactive:

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

Non-interactive:

```powershell
git governance --interactive never --output json --yes workflow ticket start `
  --family feature `
  --key ABC `
  --ticket 123 `
  --slug add-export-button `
  --scratch
```

Omit `--scratch` when the official ticket branch should be the active branch.
Use `--scratch-slug` only when the default `<slug>-exploration` name is not
appropriate.
