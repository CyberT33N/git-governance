# Hosting platforms

Hosting platforms are outside the `git-governance` domain core. The CLI owns
branch, commit, and workflow validation; each platform adapter and its
repository settings remain the final remote enforcement layer.

This tree documents **platform-specific contracts** that implement the
architectural workflow boundaries defined in
[ADR-0001](../architecture/ADR-0001-GO-CLI-ZIELARCHITEKTUR.md) and the
[workflow usage guides](../usage/workflows/index.md).

Do not redefine ticket, branch, or commit grammar here. Those remain in
[Policy and validation](../specification/POLICY-UND-VALIDIERUNG.md).

## Providers

| Provider | Status | Documents |
|---|---|---|
| [GitHub](github/develop-line.md) | required for this repository | `develop` integration-line merge policy |
| GitLab | reserved | add under `gitlab/` when a contract is adopted |
| Other hosts | reserved | add a provider directory when a contract is adopted |

## Layout

```text
docs/platforms/
  index.md                 # this catalog
  github/                  # GitHub hosting contracts
    develop-line.md        # develop MUST accept rebase-only merges
  gitlab/                  # reserved for GitLab contracts
```
