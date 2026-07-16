# Synchronize the target base

```powershell
git governance branch validate
git governance branch validate --branch feature/ABC-123-add-export-button

git governance branch sync-base --strategy check
git governance --yes branch sync-base --strategy rebase
git governance --yes branch sync-base --strategy merge `
  --merge-type chore `
  --merge-subject "merge origin/develop"
```

For automation, make the no-prompt mode explicit:

```powershell
git governance --interactive never --output json --yes branch sync-base `
  --strategy merge `
  --merge-type chore `
  --merge-subject "merge origin/develop"
```

Policy:

1. Fetch the selected remote.
2. Compare `HEAD` with the real target base.
3. If no base commits are missing, do nothing.
4. An unpublished official branch may rebase only if the base advanced.
5. A published official branch never routine-rebases; a controlled merge is
   required instead.
6. `scratch/*` remains private and is not a pull-request branch.
