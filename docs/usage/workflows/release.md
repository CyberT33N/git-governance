# Release and support workflows
## Release cut

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release cut `
  --version 2.8.0 `
  --dispatch
```

The CLI validates the requested `release/2.8.0` line and emits an intent for
the protected `create-protected-line.yml` GitHub Actions workflow. With
`--dispatch`, the configured GitHub lifecycle adapter starts that workflow,
waits for its successful correlated run, fetches the remote, and verifies that
`origin/release/2.8.0` exists. The CLI never creates, switches to, or pushes a
shared `release/*` branch itself.

Without `--dispatch`, `cut` remains an intent-only plan. It is useful for
review or a manually operated release process, but it does not prove that a
release line exists and cannot advance the governed release lifecycle.

## Stabilization

Only release-blocking fixes, final documentation, and release preparation are
allowed after a cut. Create the corresponding short-lived stabilization branch:

```powershell
git governance --yes workflow release stabilize `
  --release release/2.8.0 `
  --kind blocker `
  --key ABC `
  --ticket 999 `
  --slug release-blocker-timeout
```

Publish its pull-request intent back to the frozen release line:

```powershell
git governance --yes workflow release publish-stabilization `
  --release release/2.8.0 `
  --push
```

For non-interactive execution, use `--interactive never --output json --yes`.
After resolving and staging a paused rebase, add `--resume` to
`workflow release publish-stabilization`; the original `--release` value is
still required.

## Promotion, delivery, and conditional backmerge

After approval, create the release promotion:

```powershell
git governance workflow release promote --release release/2.8.0
```

The default output is a provider-neutral intent. To create the main pull
request through GitHub, use explicit provider configuration and confirmation:

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release promote `
  --release release/2.8.0 `
  --create-pull-request
```

Do not invoke `backmerge` merely because the promotion PR exists. It is
permitted only after the promotion merged, the immutable `v2.8.0` tag points
to that merge commit, and the required release artifacts and GitHub Release
were published successfully.

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release backmerge `
  --release release/2.8.0 `
  --create-pull-request
```

The command verifies those delivery facts with GitHub and compares
`release/2.8.0` with `develop`:

- `status=required`: it creates or returns the reviewed
  `release/2.8.0 -> develop` PR.
- `status=not-required`: no effective release-only delta remains, so it
  creates no empty PR. Record this result before release-branch cleanup.

See [release reconciliation](release-reconciliation.md) for the complete
state and evidence contract.

Use a least-privileged release-automation identity for protected-line dispatch
and delivery verification. A managed credential broker is the preferred
non-interactive mechanism; publication and dispatch never start login
implicitly. See [GitHub App authentication](../authentication.md).

After the protected `release/<semver> -> main` pull request merges, GitHub
Actions creates the immutable annotated `v<semver>` tag at that exact merge
commit. The tag workflow then dispatches the artifact workflow for that tag.
The local CLI never tags or pushes `main`.

## Support line
```powershell
git governance --yes --pull-request-provider github workflow release support `
  --version 2.8 `
  --dispatch
```

The command requires a matching `v2.8.<patch>` release tag on `origin/main`
and dispatches the same protected-line workflow. The privileged workflow
creates and the CLI verifies the remote support line from the tagged
`origin/main` revision; support lines cannot be created from an untagged
integration state.
