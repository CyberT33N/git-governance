# Release and support workflows
## Release cut

```powershell
git governance --yes workflow release cut --version 2.8.0
```

The CLI validates the requested `release/2.8.0` line and emits an intent for
the protected `create-protected-line.yml` GitHub Actions workflow. It does not
create, switch to, or push a local `release/*` branch. An authorized
release-environment workflow creates the remote line from `origin/develop`;
developers fetch it before starting controlled stabilization work.

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

## Promotion and backmerge

After approval, prepare the release promotion and later its backmerge:

```powershell
git governance workflow release promote --release release/2.8.0
git governance workflow release backmerge --release release/2.8.0
```

The default output is a provider-neutral intent. To create either pull request
through GitHub, use explicit provider configuration and confirmation:

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release promote `
  --release release/2.8.0 `
  --create-pull-request
```

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow release backmerge `
  --release release/2.8.0 `
  --create-pull-request
```

Use an existing GitHub App session for interactive local work or the managed
credential broker for automation; publication never starts login implicitly.
See [GitHub App authentication](../authentication.md).

After the protected `release/<semver> -> main` pull request merges, GitHub
Actions creates the immutable annotated `v<semver>` tag at that exact merge
commit. The tag workflow then dispatches the artifact workflow for that tag.
The local CLI never tags or pushes `main`.

## Support line
```powershell
git governance --yes workflow release support --version 2.8
```

The command requires a matching `v2.8.<patch>` release tag on `origin/main`
and emits the same protected-line workflow intent. The privileged workflow
creates the remote support line from the tagged `origin/main` revision;
support lines cannot be created from an untagged integration state.
