---
id: T-052
title: Create the adoption guide
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S005"]
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
blocked_by: ["T-048"]
trace_id: TBD
next_action: Complete; monitor future adoption feedback.
dedupe_key: docs-adoption-guide
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-052: Create The Adoption Guide

## Context

Readers need evaluation paths that match the proof they need. The guide should
not push people through identity or organization-type lanes.

## Requirements

- Create `docs/adoption-guide.html`.
- Cover safe trial, control review, operating-model, pilot, proof, and rollout
  decision routes.
- Include relevant concerns, safe first actions, and proof paths for each
  route.
- Link from homepage, harness ecosystem explainer, and README.

## Affected Files

- `docs/adoption-guide.html`
- `docs/harness-ecosystem/index.html`
- `docs/index.html`
- `README.md`

## BDD Evidence

- Scenario IDs: F-015-S005
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] Safe trial, control review, operating-model, pilot, proof, and rollout
  decision routes exist.
- [x] Each route lists concerns, safe first actions, and proof paths.
- [x] The adoption guide reinforces local ownership, guardrails, evidence, and
  canonical docs.

### Non-goals

- Sales or marketing landing page.

### Observability, docs, and regressions

- [x] Existing adoption explainer points to the new guide as the practical
  evaluation path.
