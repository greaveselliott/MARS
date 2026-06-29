# F-015: Documentation Site Information Architecture

**Feature ID:** F-015
**Goals:** G-DOCS-IA-001
**Status:** passing
**Owner:** COO with Product/Docs Maintainer

## Business Logic

The docs site is a curated human-facing layer, not the canonical source for
every internal rule.

Canonical harness docs remain the source of truth for operating model, BDD,
tickets, design decisions, validation, and generated docs.

Public docs must label canonical docs as one of:

- **Source of truth**: the durable rule or contract.
- **Used by agents**: harness or repo docs included in agent context.
- **Reference**: supporting material for operators and maintainers.

Security, guardrails, ownership, evidence, and local-first data boundaries are
first-class navigation concepts.

Every public page must enable at least one reader action:

- understand
- evaluate
- try safely
- operate
- govern
- extend
- troubleshoot
- inspect canonical records

Commands must distinguish:

- read-only inspection
- local machine writes
- target-file writes
- autonomous or agent-mediated changes

MarsDocSync metadata must connect public docs site files to the docs product
brief and existing DocSync architecture.

## Step-By-Step Behavior

1. A first-time reader lands on `docs/index.html`.
2. The first viewport says MARS is a local AI product engineering team that
   takes ideas from concept to production.
3. The reader sees trust pillars: local ownership, guardrails, evidence, human
   control, strategy loop, and performance feedback.
4. The reader chooses a path by intent and reader type.
5. Safety-conscious readers can reach security, ownership, guardrails, and
   evidence without reading CLI reference first.
6. Operators can reach quickstart, workflows, lifecycle, operations, and
   troubleshooting.
7. Maintainers and agents can reach canonical system records through the
   documentation map.
8. Each public page links to canonical harness docs when it summarizes a
   source-of-truth rule.
9. Search and filter remain useful, but the homepage no longer carries the full
   catalog.

## Scenario Schedule

### F-015-S001: Planning State Alignment

Given the docs IA goal is active
When an agent inspects the planning state
Then goal, exec plan, feature, and tickets point to the same docs IA outcome.

### F-015-S002: Trust-Building Homepage

Given a new evaluator opens the homepage
When they read the first viewport
Then they understand product team scope, local control, guardrails, and
evidence.

### F-015-S003: Documentation Map

Given a reader needs a canonical doc
When they open the documentation map
Then docs are grouped by purpose and marked as public guide, canonical harness
doc, product spec, design decision, BDD contract, validation evidence,
generated reference, or runbook.

### F-015-S004: Governance Guide

Given a security or regulatory reader opens the governance guide
When they scan the page
Then they can find data locality, credentials, guardrails, trust levels, audit
evidence, ownership, rollback or eject, and DocSync.

### F-015-S005: Adoption Guide

Given an individual, SME, enterprise, bank, or healthcare reader opens the
adoption guide
When they choose their context
Then they see relevant concerns, safe first actions, and proof paths.

### F-015-S006: Canonical Crosslinks

Given a public guide summarizes a canonical rule
When the reader reaches the summary
Then the canonical source-of-truth doc is linked and labelled.

### F-015-S007: IA Quality Gates

Given the IA changes are complete
When checks run
Then link sweep, DocSync audit, docs consistency, and full tests pass.

## Out of Scope

- CLI or runtime behavior changes.
- Generated docs generator implementation.
- Replacing the static HTML, CSS, and JavaScript stack.
- Rewriting all design docs, BDD contracts, validation reports, or ticket
  history.

## Descoped Scenarios

None.

## Evidence

- PASS: T-048 created the docs product brief and aligned active goal, active
  plan, feature catalog, product-spec catalog, and ticket state.
- PASS: T-049 rewrote `docs/index.html` as a trust-building front door.
- PASS: T-050 created `docs/documentation-map.html` and moved catalog behavior
  out of the homepage.
- PASS: T-051 created `docs/security-governance-guide.html` and crosslinked
  security, ownership, guardrail, evidence, and recovery references.
- PASS: T-052 created `docs/adoption-guide.html` with individual, SME,
  enterprise, bank/healthcare, platform, and maintainer lanes.
- PASS: T-053 crosslinked existing guides to the new IA and canonical docs.
- PASS: `git diff --check`.
- PASS: `node --check docs/site.js`.
- PASS: recursive HTML link/anchor sweep for 31 HTML files.
- PASS: primary nav consistency assertion for 30 template HTML files.
- PASS: `mars docsync audit --repo .`.
- PASS: `go test ./internal/docsconsistency ./internal/docsync`.
- PASS: `go test ./...`.
