# Automate release workflows

Use the same provider-neutral defaults in automation. `--interactive never`
requires every mandatory value as a flag; `--yes` authorizes the bounded
mutation or protected-line request.

```powershell
git governance --interactive never --output json --yes workflow release cut `
  --version 2.8.0

git governance --interactive never --output json --yes workflow release stabilize `
  --release release/2.8.0 `
  --kind blocker `
  --key ABC `
  --ticket 999 `
  --slug release-blocker-timeout

git governance --interactive never --output json --yes workflow release publish-stabilization `
  --release release/2.8.0 `
  --push

git governance --interactive never --output json --yes workflow release promote `
  --release release/2.8.0

git governance --interactive never --output json --yes workflow release backmerge `
  --release release/2.8.0

git governance --interactive never --output json --yes workflow release support `
  --version 2.8
```

For an actual GitHub PR during stabilization, promotion, or backmerge, add
`--pull-request-provider github --create-pull-request`. Stabilization also
requires `--push`; promotion and backmerge do not.
