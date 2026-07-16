# Getting started

## Prerequisites for contributors

- Git 2.53 or newer recommended
- Go 1.26.5 (enforced by the `toolchain go1.26.5` directive)

Check the local environment:

```powershell
go version
git --version
```

Expected Go output begins with:

```text
go version go1.26.5
```

## Build from source

Clone or open the repository, then run the full build gate:

```powershell
go run .\cmd\build
```

On macOS or Linux:

```bash
go run ./cmd/build
```

`cmd/build` verifies root and build-tool module integrity, checks formatting,
runs Staticcheck, typechecks packages and tests, runs unit, contract,
integration, coverage, race, vet, vulnerability, fuzz, and Lefthook checks,
then builds and smoke-tests the native binary. It stops at the first failed
gate and writes `dist\git-governance.exe` on Windows or
`dist/git-governance` on macOS and Linux.

For a local development run without producing a binary:

```powershell
go run .\cmd\git-governance --help
```

To use the Git subcommand form locally, put the built binary in a directory
already on `PATH`:

```powershell
git governance --help
```

Release installers and package-manager manifests are added by the release
pipeline. They are not yet published by this repository.
