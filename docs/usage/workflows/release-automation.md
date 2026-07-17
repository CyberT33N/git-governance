# Automate release workflows

Use the same provider-neutral defaults in automation. `--interactive never`
requires every mandatory value as a flag; `--yes` authorizes the bounded
mutation or protected-line request.

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release cut `
  --version 2.8.0 `
  --dispatch

git governance --interactive never --output json --yes workflow release stabilize `
  --release release/2.8.0 `
  --kind blocker `
  --key ABC `
  --ticket 999 `
  --slug release-blocker-timeout

git governance --interactive never --output json --yes workflow release publish-stabilization `
  --release release/2.8.0 `
  --push

git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release promote `
  --release release/2.8.0 `
  --create-pull-request

git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release backmerge `
  --release release/2.8.0 `
  --create-pull-request

git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release support `
  --version 2.8 `
  --dispatch
```

For an actual GitHub PR during stabilization, promotion, or a required
backmerge, add `--create-pull-request`. Stabilization also requires `--push`;
promotion and backmerge do not. `cut --dispatch` and `support --dispatch`
wait for protected-line creation and verify the resulting remote line.

Backmerge must run only after the main promotion, exact immutable tag, and
release artifact delivery are complete. It returns `not-required` instead of
creating an empty PR when no effective release-only delta remains. Automation
must use a managed credential broker or another least-privileged release
identity; it never starts a browser login. See
[GitHub App authentication](../authentication.md).
