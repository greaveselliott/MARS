---
id: T-049
title: Rebuild the docs homepage IA
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S002"]
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
next_action: Complete; monitor future homepage feedback.
dedupe_key: docs-homepage-trust-front-door
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-049: Rebuild The Docs Homepage IA

## Context

The homepage should introduce MARS as a complete local AI product engineering
team, then route readers by intent. It should not carry the full documentation
catalog.

## Requirements

- Hero headline: "MARS is a local AI product engineering team you can inspect,
  govern, and improve."
- Show trust pillars: local ownership, guardrails, evidence, human control,
  strategy loop, and performance feedback.
- Show reader paths for individual, SME team, enterprise, bank or healthcare,
  operator, and maintainer/agent.
- Separate read-only actions, local writes, target writes, and autonomous
  actions.
- Explain that public docs summarize and canonical harness docs are source of
  truth where labelled.

## Affected Files

- `docs/index.html`
- `docs/site.css`
- `docs/site.js`

## BDD Evidence

- Scenario IDs: F-015-S002
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] First viewport explains team scope, local control, guardrails, and
  evidence.
- [x] Full catalog behavior is moved out of the homepage.
- [x] Homepage supports mobile viewports through existing responsive template
  and new layout classes.

### Non-goals

- Framework migration.

### Observability, docs, and regressions

- [x] `docs/site.css` and `docs/site.js` DocSync metadata includes the new IA
  pages and product spec where affected.
