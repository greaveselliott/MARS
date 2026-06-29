---
id: T-050
title: Create the documentation map
priority: high
complexity: medium
kind: standard
work_type: docs
bdd_scenarios: ["F-015-S003"]
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
next_action: Complete; keep map current as docs move or new docs are added.
dedupe_key: docs-documentation-map
source: user_chat docs IA rebuild 2026-06-29
created: 2026-06-29
---

# T-050: Create The Documentation Map

## Context

The homepage should not be the complete catalog. Readers still need one place
to find canonical MARS documents by purpose and source-of-truth status.

## Requirements

- Create `docs/documentation-map.html`.
- Link it from the homepage and README.
- Group docs by: start and safe trial, daily operation, security/governance,
  agent team and delivery model, extension/integration, canonical system
  records, validation/evidence, and generated/reference material.
- Each entry includes path, purpose, headline, audience, doc type, canonical
  status, and "used by agents?".

## Affected Files

- `docs/documentation-map.html`
- `docs/index.html`
- `README.md`

## BDD Evidence

- Scenario IDs: F-015-S003
- Evidence links: `git diff --check`; `node --check docs/site.js`; HTML link/anchor sweep; primary nav consistency; `mars docsync audit --repo .`; `go test ./internal/docsconsistency ./internal/docsync`; `go test ./...`
- Verified by: foundation-maintainer; docs static checks; DocSync; docs consistency; full Go suite

## Acceptance Criteria

### Functional

- [x] Canonical docs are marked clearly.
- [x] Public guides and canonical harness docs are separated by purpose.
- [x] Search/filter can find entries by path, purpose, audience, and type.

### Non-goals

- Moving existing URLs.

### Observability, docs, and regressions

- [x] Documentation map links remain relative and valid from GitHub Pages.
