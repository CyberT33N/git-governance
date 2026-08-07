# Changelog: Governed Task-to-PR Workflow Prompt
[INTENT: REFERENCE]

---

## 1. Scope Metadata
[INTENT: CONTEXT]

| Field | Value |
|-------|-------|
| Scope Root | `__prompts__/version-control/git/workflow` |
| Versioning Standard | `Semantic Versioning 2.0.0` |
| Current Version | `1.0.0` |
| Semver Class | `patch` |
| Breaking Change | `no` |
| Commit Scope | `GOV-35` |
| Current HEAD Commit Hash | `b12a93a741e39ef53d3851113a16db93e2831dcf` |

---

## 2. Version Ledger
[INTENT: REFERENCE]

| Version | Date | Class | Breaking | Commit Type | HEAD Commit Hash | Summary | Commit Subject |
|---------|------|-------|----------|-------------|------------------|---------|----------------|
| `1.0.0` | `2026-08-07` | `patch` | `no` | `docs` | `b12a93a741e39ef53d3851113a16db93e2831dcf` | Canonicalized the governed workflow prompt and its stable Cursor entrypoint. | `docs(GOV-35): centralize canonical rule prompts` |

---

## 3. Current Version Entry
[INTENT: SPECIFICATION]

### 3.1 Version `1.0.0`
[INTENT: SPECIFICATION]

**Classification**

| Field | Value |
|-------|-------|
| Semver Class | `patch` |
| Breaking Change | `no` |
| Rationale | The complete governed workflow content is preserved; only its canonical source location, relative Cursor entrypoint, and metadata pair are introduced. |

**Change Units**

| ID | Category | Breaking | Summary | Affected Files | Description Alignment |
|----|----------|----------|---------|----------------|----------------------|
| CHG-001 | policy | no | Added the complete canonical workflow prompt and retained the Cursor rule through a relative symbolic link. | `prompt.md`, `.cursor/rules/governed-task-to-pr-workflow.mdc` | `REQ-001`, `REQ-002` |
| CHG-002 | docs | no | Added the canonical description and changelog metadata pair. | `DESCRIPTION.md`, `CHANGELOG.md` | `META-001` |

**Migration / Consumer Impact**

No migration required.

**Commit Alignment**

| Field | Value |
|-------|-------|
| Commit Subject | `docs(GOV-35): centralize canonical rule prompts` |
| Breaking Footer | `none` |
| Current HEAD Commit Hash | `b12a93a741e39ef53d3851113a16db93e2831dcf` |

---

## 4. Semver Rules
[INTENT: REFERENCE]

- Patch = non-breaking correction, clarification, or metadata synchronization.
- Minor = backward-compatible addition.
- Major = breaking contract or incompatible architectural change.

---

## 5. Path Index
[INTENT: REFERENCE]

| # | Path | Relevance |
|---|------|-----------|
| 1 | `prompt.md` | Canonical governed workflow source |
| 2 | `DESCRIPTION.md` | Scope reference |
| 3 | `CHANGELOG.md` | Version ledger |
| 4 | `.cursor/rules/governed-task-to-pr-workflow.mdc` | Relative Cursor rule entrypoint |
