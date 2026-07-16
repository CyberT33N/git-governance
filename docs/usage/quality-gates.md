# Project-agnostic quality gates

The CLI does not guess a project's language, package manager, build command,
or linter. Hardcoding `go test`, `npm test`, `mvn test`, or any other stack
command would make a generic Git tool architecturally incorrect.

Instead, a trusted repository can opt in with
`git-governance.quality.json` at its root, or an explicit
`--quality-config <path>`. Commands are executed as an executable plus argument
array; no shell command string is interpreted.

```json
{
  "schemaVersion": 2,
  "defaults": {
    "includeFamilies": [
      "feature", "fix", "docs", "refactor",
      "chore", "test", "perf", "hotfix"
    ]
  },
  "gates": [
    {
      "name": "unit-tests",
      "command": "go",
      "args": ["test", "./..."],
      "timeout": "2m"
    },
    {
      "name": "documentation-links",
      "command": "npm",
      "args": ["run", "docs:check"],
      "includeFamilies": ["docs"],
      "timeout": "2m"
    },
    {
      "name": "stress",
      "command": "./scripts/stress-test",
      "includeFamilies": ["feature", "perf"],
      "timeout": "2m"
    }
  ]
}
```

Each gate uses a repository-relative working directory and a positive Go
duration timeout. Paths that escape the repository, shell control characters,
unknown JSON fields, duplicate gate names, and unbounded configuration are
rejected.

If no configuration exists, the workflow reports `qualityStatus:
unconfigured`; it does not claim that project-specific checks passed. The
configuration is a trust boundary because running project-defined commands can
execute project code. Review it before using it in an unfamiliar repository.

When a valid configuration exists, each gate receives a typed branch-family
scope. A gate without `includeFamilies` or `excludeFamilies` inherits
`defaults`. `includeFamilies` selects only the listed families;
`excludeFamilies` is applied afterward and removes specific families. A
multi-ref push runs every eligible gate once after its per-ref governance
checks pass.

The recommended default includes every official working family:
`feature`, `fix`, `docs`, `refactor`, `chore`, `test`, `perf`, and `hotfix`.
`scratch` is absent from that default because it is private exploration. It is
not a hardcoded exception: a repository can include `scratch` for one small
gate without enabling expensive gates there. This lets a documentation branch
run link checks while skipping a stress test, or a performance branch run a
stress test that other families do not need.
