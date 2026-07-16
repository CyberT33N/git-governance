# Lefthook integration

Install Lefthook through your approved package channel, make
`git-governance` available on `PATH`, then run:

```powershell
lefthook install
lefthook validate
```

The repository `lefthook.yml` is intentionally thin:

- `commit-msg` runs `git-governance commit validate --message-file "{1}"`;
- `pre-push` runs `git-governance validate pre-push` and forwards Git stdin.

No branch regex, ticket policy, network call, rebase, merge, or direct Git
mutation is duplicated in Lefthook configuration.

The `pre-push` validator parses every update supplied by Git, including
`HEAD:main`, multiple ref updates, deletion requests, and non-fast-forward
updates. Shared-line mutations and official-branch force pushes are blocked
against the actual remote target rather than the checked-out branch.
