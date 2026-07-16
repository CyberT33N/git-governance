# Ticket-key preferences

Keys are convenience preferences, not organizational authorization. v1 accepts
every syntactically valid key:

```regex
^[A-Z][A-Z0-9]*$
```

Manage keys:

```powershell
git governance config key add --key PLATFORM2
git governance config key set-default --key PLATFORM2
git governance config key list
git governance config key remove --key PLATFORM2
```

All key commands are scriptable:

```powershell
git governance --interactive never --output json config key add --key PLATFORM2
git governance --interactive never --output json config key set-default --key PLATFORM2
git governance --interactive never --output json config key list
git governance --interactive never --output json config key remove --key PLATFORM2
```

Configuration location uses Go `os.UserConfigDir()`:

| Platform | Default application directory |
|---|---|
| Linux | `$XDG_CONFIG_HOME/git-governance` or `$HOME/.config/git-governance` |
| macOS | `$HOME/Library/Application Support/git-governance` |
| Windows | `%AppData%\git-governance` |

The file is versioned JSON and uses a platform-aware recoverable replacement
strategy. Unix replacement uses same-directory rename semantics; Windows keeps
and restores a temporary `.bak` recovery copy if an interrupted replacement
leaves the target absent. It never stores secrets or a global default ticket
number.

GitHub tokens are separate deployment configuration. Set
`GIT_GOVERNANCE_GITHUB_TOKEN` in the invoking environment; never place it in
the preference file or a command-line flag.

For GitHub Enterprise, set `GIT_GOVERNANCE_GITHUB_API_URL` to the HTTPS API
root. Public GitHub defaults to [api.github.com](https://api.github.com).
