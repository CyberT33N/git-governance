# Publish ticket work

Publish after development:

```powershell
git governance --yes workflow ticket publish --push
```

When invoked from an official ticket branch, publication follows the normal
flow. When invoked from a `scratch/*` branch, the workflow first resolves the
same local official ticket branch, shows both branch names and the supplied
squash commit, and asks for confirmation. After confirmation it reuses the
same `branch merge-scratch` application component before validating and
optionally pushing the official branch:

```powershell
git governance workflow ticket publish `
  --type feat `
  --subject "add export button" `
  --push
```

The workflow validates the commit series, runs configured quality gates,
checks base freshness, conditionally rebases only an unpublished branch, then
revalidates branch policy, the full commit series, and quality gates before it
can push. Interactive publication reports whether a rebase happened or why it
did not. After a push, it asks before creating a pull request only when a
hosting-provider adapter is configured. Without such an adapter, it reports
the provider-neutral pull-request intent targeting `develop`.

## Non-interactive publication

For non-interactive execution, use `--interactive never --yes` and provide
the commit family and description when publishing from scratch. `--message`
remains available only as the complete-message compatibility input. Use
`--target <official-branch>` only when a manually created repository has more
than one local official branch for the ticket.

```powershell
git governance --interactive never --output json --yes workflow ticket publish `
  --push
```

```powershell
git governance --interactive never --output json --yes workflow ticket publish `
  --type feat `
  --subject "add export button" `
  --push
```

To create the GitHub pull request as part of the same explicit operation, log
in first with `auth login github` or configure the managed credential broker,
then set the provider and add `--create-pull-request`:

```powershell
git governance --interactive never --output json --yes `
  --pull-request-provider github workflow ticket publish `
  --push `
  --create-pull-request
```

The adapter derives the GitHub host, owner, and repository from the selected
remote. It resolves a short-lived GitHub App credential just in time, verifies
the exact App/user or broker/repository authorization, checks for an equivalent
open pull request, and creates one only when none exists.
`--create-pull-request` requires `--push`; the default remains an intent-only
result. Publication never starts browser login or accepts a GitHub token flag;
see [GitHub App authentication](../../authentication.md).
