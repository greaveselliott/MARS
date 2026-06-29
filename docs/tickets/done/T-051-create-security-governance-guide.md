---
id: T-051
title: Create the security and governance guide
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S004"]
end_to_end_evidence: not_applicable
evidence_links:
  - "PASS: git diff --check"
  - "PASS: node --check docs/site.js"
  - "PASS: HTML link/anchor sweep checked 31 files"
  - "PASS: primary nav consistency checked 30 template pages"
  - "PASS: mars docsync audit --repo ."
  - "PASS: go test ./internal/docsconsistency ./internal/docsync"
  - "PASS: go test ./..."
verified_by: "foundation-maintainer; security/governance role packet; docs static checks; DocSync; docs consistency; full Go suite"
owner: foundation-maintainer
last_attempt: "2026-06-29"
blocker: none
blocked_by: ["T-048"]
trace_id: TBD
next_action: Complete; monitor future security/regulatory feedback.
dedupe_key: docs-security-governance-guide
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-051: Create The Security And Governance Guide

## Context

Security, enterprise, bank, healthcare, and regulated readers need one plain
guide that routes them to canonical evidence without overstating guarantees.

## Requirements

- Create `docs/security-governance-guide.html`.
- Cover what stays local, credential boundaries, guardrails, role allowlists,
  trust levels, ownership of generated work, evidence chain, logs/traces/checks,
  pause/stop/eject/rollback, and DocSync.
- Link canonical source docs for every summarized rule.

## Affected Files

- `docs/security-governance-guide.html`
- `docs/safety-quality-guide.html`
- `docs/auth-credentials-reference.html`
- `docs/files-state-reference.html`
- `docs/guardrails-reference.html`
- `docs/observability-guide.html`

## BDD Evidence

- Scenario IDs: F-015-S004
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; security/governance role packet; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] Regulated readers can find data locality, credentials, guardrails, trust
  levels, audit evidence, ownership, rollback/eject, and DocSync.
- [x] Claims are qualified: local-first by default, opt-in integrations and
  cloud routes, syntactic guardrails where applicable, and auditability rather
  than deterministic LLM behavior.

### Non-goals

- New security controls or compliance certification.

### Observability, docs, and regressions

- [x] Existing security-related public guides link back to the governance guide.
