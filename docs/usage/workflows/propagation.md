# Propagate a hotfix

Forward-port or backport one reviewed hotfix commit. The command creates a
short-lived `fix/*` branch from the requested target line, applies
`git cherry-pick -x`, validates it, and emits a pull-request intent:

```powershell
git governance --yes workflow hotfix propagate `
  --target-line develop `
  --commit 0123456789abcdef0123456789abcdef01234567 `
  --push
```

Non-interactive automation uses the same explicit inputs:

```powershell
git governance --interactive never --output json --yes workflow hotfix propagate `
  --source hotfix/ABC-999-payment-timeout `
  --target-line develop `
  --commit 0123456789abcdef0123456789abcdef01234567 `
  --push
```

Add `--pull-request-provider github --create-pull-request` to publish the
resulting PR after its branch is pushed.

If a cherry-pick pauses for conflicts, resolve and stage it, then resume the
already-created propagation branch:

```powershell
git governance --interactive never --output json --yes workflow hotfix propagate `
  --source hotfix/ABC-999-payment-timeout `
  --target-line develop `
  --branch fix/ABC-999-forward-port-payment-timeout `
  --resume `
  --push
```
