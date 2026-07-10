# Package-manager publication templates

`git-governance` publishes immutable GitHub Release artifacts first. The
templates in this directory are then filled from that release's version and
SHA-256 checksum manifest.

They are deliberately templates rather than active publication configuration:

- Homebrew requires a maintainer-controlled tap repository.
- Scoop requires a maintainer-controlled bucket repository.
- WinGet requires a package identifier and a submitted manifest in the public
  winget-pkgs repository.
- Windows Authenticode and macOS signing/notarization require publisher
  certificates and service credentials that must never be committed here.

No external repository, credential, package endpoint, or publisher identity is
invented by this project. A release may publish the signed GitHub artifacts and
Linux packages without these optional channels. Turning on a package-manager
channel requires a reviewed maintainer-owned destination and a successful
package-manager validation run.

## Release substitution contract

Replace these placeholders only from the final immutable release:

- `{{VERSION}}`: version without the leading `v`, for example `2.8.0`
- `{{TAG}}`: Git tag with the leading `v`, for example `v2.8.0`
- `{{*_SHA256}}`: matching entry from
  `git-governance_{{VERSION}}_checksums.txt`

The release process must verify the source release asset and its checksum
before submitting any template. Do not replace a published package version or
checksum in place.

## Artifact-name contract

The templates use the archive names emitted by `.goreleaser.yaml`:

```text
git-governance_{{VERSION}}_windows_amd64.zip
git-governance_{{VERSION}}_windows_arm64.zip
git-governance_{{VERSION}}_darwin_amd64.tar.gz
git-governance_{{VERSION}}_darwin_arm64.tar.gz
git-governance_{{VERSION}}_linux_amd64.tar.gz
git-governance_{{VERSION}}_linux_arm64.tar.gz
```

`windows`, `darwin`, `linux`, `amd64`, and `arm64` are GoReleaser's Go target
identifiers. Package templates must not substitute display names such as
`Windows`, `Darwin`, or `x86_64`, because those strings do not name published
assets.
