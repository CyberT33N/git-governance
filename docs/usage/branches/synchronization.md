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

1. Run the command on the branch to synchronize. If supplied, `--branch` must
   match the checked-out branch; it never switches branches implicitly.
2. Fetch the selected remote.
3. Compare `HEAD` with the real target base.
4. If no base commits are missing, do nothing.
5. An unpublished official branch may rebase only if the base advanced.
6. A published official branch never routine-rebases; a controlled merge is
   required instead.
7. `scratch/*` remains private and is not a pull-request branch.
8. A release-preparation branch must not use this generic command to import
   `main` for a release promotion. Use
   `workflow release align-promotion-base` so the release provenance,
   quality gate, and release-line PR target remain explicit.
