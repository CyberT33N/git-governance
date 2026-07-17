# Overview

`git-governance` is a native Go CLI for governed Git work on Windows, macOS,
and Linux. It creates and validates branches and commits, guides bounded
ticket workflows, and exposes the same validation core to Lefthook and CI.

The binary is named `git-governance`. When it is on `PATH`, Git also exposes it
as:

```text
git governance ...
```

No Go, Node.js, Python, PowerShell, or shell runtime is required on end-user
machines once a release binary is installed. Git itself is required.

## What the tool owns

- canonical branch, ticket, slug, SemVer, and Conventional Commit validation
- branch creation from the correct remote-tracking base
- commit creation with explicit staging only
- bounded ticket, hotfix, release, support, and delivery-gated conditional
  backmerge workflows
- first-push base-freshness checks
- stable human and JSON error contracts
- known ticket-key preferences in the operating-system configuration directory
- local Lefthook integration through the same validator commands

The tool does not replace Git hosting, branch protection, CI, a ticket tracker,
or a policy registry.

## Architecture

```text
Cobra / Huh / JSON
        ↓
Application use cases
        ↓
Branch, ticket, and commit domain models
        ↓
Application-owned ports
        ↓
Git CLI, config filesystem, terminal, reporter, and hosting adapters
```

The domain never depends on Cobra, Huh, Git process APIs, environment
variables, filesystem paths, or a hosting provider.
