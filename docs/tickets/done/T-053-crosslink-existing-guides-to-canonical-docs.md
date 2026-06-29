---
id: T-053
title: Crosslink existing guides to canonical docs
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S006", "F-015-S007"]
end_to_end_evidence: not_applicable
evidence_links:
  - "PASS: git diff --check"
  - "PASS: node --check docs/site.js"
  - "PASS: HTML link/anchor sweep checked 31 files"
  - "PASS: primary nav consistency checked 30 template pages"
  - "PASS: mars docsync audit --repo ."
  - "PASS: go test ./internal/docsconsistency ./internal/docsync"
  - "PASS: go test ./..."
verified_by: "foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite"
owner: foundation-maintainer
last_attempt: "2026-06-29"
blocker: none
blocked_by: ["T-049", "T-050", "T-051", "T-052"]
trace_id: TBD
next_action: Complete; monitor future docs consistency and UX feedback.
dedupe_key: docs-existing-guide-canonical-crosslinks
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-053: Crosslink Existing Guides To Canonical Docs

## Context

Public guides can summarize canonical behavior, but they must not become a
parallel source of truth. Readers need clear labels and routes back to system
records.

## Requirements

- Add "Source of truth" or "Used by agents" callouts where pages summarize
  canonical behavior.
- Ensure each touched page answers: what this is, when to use it, what changes,
  how to check it worked, and how to recover.
- Add links back to `documentation-map.html`,
  `security-governance-guide.html`, and `adoption-guide.html` where relevant.
- Run final docs IA verification gates.

## Affected Files

- `docs/quickstart.html`
- `docs/workflows.html`
- `docs/planning-delivery-guide.html`
- `docs/documentation-sync-guide.html`
- `docs/target-lifecycle-reference.html`
- `docs/harness-guide.html`
- `docs/tools-mcp-guide.html`
- `docs/models-guide.html`
- `docs/release-update-guide.html`
- `docs/integrations-validation-guide.html`

## BDD Evidence

- Scenario IDs: F-015-S006, F-015-S007
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] Existing guides link to canonical docs when they summarize source-of-truth
  rules.
- [x] Existing guides route readers to documentation map, security/governance,
  and adoption pages where useful.
- [x] Public summaries are labelled as summaries and avoid duplicating canonical
  docs silently.

### Non-goals

- Rewriting every public guide in full.

### Observability, docs, and regressions

- [x] `git diff --check` passes.
- [x] `node --check docs/site.js` passes.
- [x] Recursive HTML link/anchor sweep passes.
- [x] Primary nav consistency check passes.
- [x] `mars docsync audit --repo .` passes.
- [x] `go test ./internal/docsconsistency ./internal/docsync` passes.
- [x] `go test ./...` passes.
