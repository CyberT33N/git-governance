# Diagnostics, policy, errors, and completions

```powershell
git governance doctor
git governance --output json policy describe
git governance completion powershell > git-governance.ps1
```

Generate scripts for `bash`, `zsh`, `fish`, or `powershell`.

`doctor` is read-only. It reports the Git version, repository and history
state, selected remote availability, active rebase/merge/cherry-pick state,
Lefthook binary/configuration status, local policy mode, and user
configuration status.

## Errors and exit codes

All errors have a stable code, field, non-sensitive actual value, rule, valid
example, and remediation. Sensitive values are redacted in both human and JSON
output. In JSON mode, errors have this shape:

```json
{
  "schemaVersion": 1,
  "ok": false,
  "error": {
    "code": "COMMIT_TICKET_MISMATCH",
    "category": "governance",
    "field": "ticket"
  }
}
```

Exit codes:

| Code | Meaning |
|---:|---|
| 0 | success |
| 1 | internal failure |
| 2 | CLI usage or missing input |
| 3 | governance or policy violation |
| 4 | repository state failure |
| 5 | Git operation failure |
| 6 | configuration failure |
| 7 | external adapter failure |
| 130 | cancellation |

## Further documentation

- [Architecture Decision Record](../architecture/ADR-0001-GO-CLI-ZIELARCHITEKTUR.md)
- [Policy and validation contract](../specification/POLICY-UND-VALIDIERUNG.md)
- [CLI contract](../specification/CLI-VERTRAG.md)
- [Installation and release design](../operations/INSTALLATION-UND-RELEASE.md)
- [Product acceptance matrix](../TRACEABILITY.md)
- [Contributing](../../CONTRIBUTING.md)
- [Package-manager publication templates](../../packaging/README.md)
