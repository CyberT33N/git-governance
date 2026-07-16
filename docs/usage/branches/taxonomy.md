# Branch taxonomy

List every supported family and its purpose:

```powershell
git governance branch list
git governance --output json branch list
```

The complete taxonomy is:

| Family | Name pattern | Purpose |
|---|---|---|
| `main` | `main` | published production truth |
| `develop` | `develop` | integration line |
| `release` | `release/<semver>` | frozen release candidate |
| `support` | `support/<major.minor>` | maintained older version line |
| `feature` | `feature/<ticket>-<slug>` | new capability |
| `fix` | `fix/<ticket>-<slug>` | regular defect correction |
| `docs` | `docs/<ticket>-<slug>` | documentation |
| `refactor` | `refactor/<ticket>-<slug>` | internal restructuring |
| `chore` | `chore/<ticket>-<slug>` | maintenance or tooling |
| `test` | `test/<ticket>-<slug>` | test work |
| `perf` | `perf/<ticket>-<slug>` | performance work |
| `hotfix` | `hotfix/<ticket>-<slug>` | defect on an active line |
| `scratch` | `scratch/<ticket>-<slug>` | private exploration |

`feature` is a branch family. `feat` is a commit type and is intentionally not
a valid branch family.
