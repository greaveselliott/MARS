---
id: T-048
title: Define the documentation site product brief
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S001"]
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
blocked_by: []
trace_id: TBD
next_action: Complete; monitor future docs IA feedback.
dedupe_key: docs-site-product-brief
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-048: Define The Documentation Site Product Brief

## Context

G-DOCS-IA-001 and F-015 reframe the documentation site as a trust-building
product documentation layer over canonical harness docs.

## Requirements

- Create `docs/product-specs/documentation-site.md`.
- Update product-spec and feature indexes.
- Align `docs/goals/active.md` and the active exec plan with F-015.
- Define evaluation routes, reader actions, content model, risks, and
  MarsDocSync relationship.

## Affected Files

- `docs/product-specs/documentation-site.md`
- `docs/product-specs/index.md`
- `docs/product-specs/product-surface.md`
- `docs/features/F-015-documentation-site-information-architecture.md`
- `docs/features/README.md`
- `docs/goals/active.md`
- `docs/exec-plans/active/current-operating-plan.md`

## BDD Evidence

- Scenario IDs: F-015-S001
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] Evaluation routes cover safe trial, fit evaluation, governed autonomy,
  operations and recovery, extension and integration, and canonical record
  inspection.
- [x] Reader actions are explicit: understand, evaluate, try safely, operate,
  govern, extend, troubleshoot, and inspect canonical records.
- [x] Content model defines public guide, public reference, canonical harness
  doc, product spec, design decision, BDD contract, validation evidence,
  generated reference, and runbook.
- [x] MarsDocSync relationship is defined.

### Non-goals

- Runtime or CLI behavior changes.

### Observability, docs, and regressions

- [x] Planning state and product spec link to F-015 and G-DOCS-IA-001.
- [x] Docs consistency covers the new product spec.
