# Description: Governed Task-to-PR Workflow Prompt
[INTENT: CONTEXT]

This directory owns the canonical, complete prompt for the repository's
governed task-to-pull-request workflow. The Cursor rule entrypoint remains
available through a relative symlink; the workflow instruction content is
preserved without semantic changes.

---

## 1. Scope Overview
[INTENT: CONTEXT]

The canonical source is `prompt.md`. It contains the full governed workflow
for branch context, source-CLI help reanchoring, intake, verification,
semantic commits, publication, release handling, and hotfix delivery gates.

The repository-level Cursor rule
`.cursor/rules/governed-task-to-pr-workflow.mdc` resolves to this source with a
relative symlink. This keeps the existing Cursor rule location stable while
making the prompt directory the single editable source.

---

## 2. Information Register
[INTENT: REFERENCE]

| ID | Type | Description | Change | Status |
|----|------|-------------|--------|--------|
| REQ-001 | REQUIREMENT | Canonicalize the complete governed workflow prompt at `prompt.md`. | Yes | Active |
| REQ-002 | REQUIREMENT | Keep the existing Cursor rule entrypoint as a relative symlink to the canonical prompt. | Yes | Active |
| CONV-001 | CONSTRAINT | Preserve the governed workflow text and its operational contract without semantic changes. | No | Active |
| META-001 | INFORMATION | Maintain canonical description and changelog metadata in this prompt directory. | Yes | Active |

---

## 3. Information Units
[INTENT: SPECIFICATION]

### 3.1 REQ-001: Canonical governed workflow source
[INTENT: SPECIFICATION]

**Type:** REQUIREMENT

**Description:**
`prompt.md` is the canonical source for the complete Governed Task-to-PR
Workflow prompt.

**Current State:**
Before this change, the complete workflow existed only as a Cursor rule under
`.cursor/rules`.

**Target State:**
The complete workflow is preserved in `prompt.md` without changing its
governance, testing, publication, release, or hotfix requirements.

**Affected Files:**

| Path | Relevance | Elements |
|------|-----------|----------|
| `__prompts__/version-control/git/workflow/prompt.md` | Canonical prompt source | Complete governed workflow |

**Positive Example(s):**

```text
__prompts__/version-control/git/workflow/prompt.md
```

is edited as the single canonical source of the governed workflow.

**Negative Example(s):**

```text
Two independently edited copies of the governed workflow
```

This would allow the Cursor rule and canonical prompt to diverge.

---

### 3.2 REQ-002: Stable Cursor rule entrypoint
[INTENT: SPECIFICATION]

**Type:** REQUIREMENT

**Description:**
The existing Cursor rule path remains available as a relative symlink to the
canonical prompt.

**Current State:**
The repository consumed the governed workflow directly from
`.cursor/rules/governed-task-to-pr-workflow.mdc`.

**Target State:**
The same path resolves to `prompt.md` through the relative target
`../../__prompts__/version-control/git/workflow/prompt.md`.

**Affected Files:**

| Path | Relevance | Elements |
|------|-----------|----------|
| `.cursor/rules/governed-task-to-pr-workflow.mdc` | Stable Cursor entrypoint | Relative symbolic link |
| `__prompts__/version-control/git/workflow/prompt.md` | Canonical target | Governed workflow text |

**Positive Example(s):**

```text
.cursor/rules/governed-task-to-pr-workflow.mdc
  -> ../../__prompts__/version-control/git/workflow/prompt.md
```

**Negative Example(s):**

```text
.cursor/rules/governed-task-to-pr-workflow.mdc
  -> C:\absolute\machine-specific\path\prompt.md
```

An absolute target would not remain portable across repository checkouts.

---

### 3.3 CONV-001: Content preservation
[INTENT: CONSTRAINT]

The canonical prompt and the Cursor rule target must remain byte-equivalent
workflow content. The migration changes source ownership and lookup only; it
does not relax, replace, or reinterpret the governed workflow contract.

---

### 3.4 META-001: Local metadata pair
[INTENT: INFORMATION]

`DESCRIPTION.md` explains the canonical prompt boundary, and `CHANGELOG.md`
records versioned metadata changes for this prompt directory.

---

## 4. Conventions and Constraints
[INTENT: CONSTRAINT]

- `prompt.md` is the canonical editable source.
- The Cursor rule path is retained as a relative symbolic link.
- The symbolic-link target stored by Git uses forward slashes and is relative
  to `.cursor/rules`.
- The governed workflow text remains complete and unchanged by this
  canonicalization.
- `DESCRIPTION.md` and `CHANGELOG.md` use their exact uppercase filenames.

---

## 5. Path Index
[INTENT: REFERENCE]

| # | Path | Relevance | Unit IDs |
|---|------|-----------|----------|
| 1 | `__prompts__/version-control/git/workflow/prompt.md` | Canonical governed workflow source | REQ-001, REQ-002, CONV-001 |
| 2 | `__prompts__/version-control/git/workflow/DESCRIPTION.md` | Scope reference for the canonical prompt | META-001 |
| 3 | `__prompts__/version-control/git/workflow/CHANGELOG.md` | Version ledger for metadata changes | META-001 |
| 4 | `.cursor/rules/governed-task-to-pr-workflow.mdc` | Relative Cursor rule entrypoint | REQ-002 |

---

## 6. Execution Context for LLM Agents
[INTENT: CONTEXT]

Read `prompt.md` as the complete governed workflow contract. The
`.cursor/rules` path is a compatibility entrypoint and must not be maintained
as a separate content source. When this prompt changes, retain the relative
link target and update this metadata pair in the same scoped change.
