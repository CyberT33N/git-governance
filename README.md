# git-governance

`git-governance` is a native Go CLI for governed Git work on Windows, macOS,
and Linux. It creates and validates branches and commits, guides bounded
ticket workflows, and exposes the same validation core to Lefthook and CI.

Start with the [documentation index](docs/index.md), then use the
[CLI usage guide](docs/usage/index.md) for complete interactive and
non-interactive command contracts.

## Command catalog

- `branch list`, `branch create`, `branch validate`, `branch merge-scratch`,
  and `branch sync-base`
- `commit create` and `commit validate`
- `workflow ticket start` and `workflow ticket publish`
- `workflow hotfix start`, `workflow hotfix publish`, and
  `workflow hotfix propagate`
- `workflow release cut`, `workflow release stabilize`,
  `workflow release publish-stabilization`, `workflow release promote`,
  `workflow release backmerge`, and `workflow release support`
- `workflow cleanup`
- `validate pre-push`
- `auth login github`, `auth status github`, and `auth logout github`
- `config key list`, `config key add`, `config key remove`, and
  `config key set-default`
- `policy describe`, `doctor`, and `completion <shell>`

For automation, use `--interactive never --output json`, supply every required
value as a flag, and add `--yes` for mutations. GitHub pull-request creation is
an explicit opt-in through `--pull-request-provider github` and
`--create-pull-request`.

GitHub API access uses an explicit GitHub App login or a managed credential
broker; it never accepts a GitHub token as a CLI flag or stores one in user
preferences. Read the [GitHub App authentication guide](docs/usage/authentication.md)
before requesting pull-request creation.
