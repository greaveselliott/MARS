# Completed P1 Exec Plan: Documentation Site IA Rebuild

**Status:** Completed
**Priority:** P1
**Depends On:** None
**Blocks:** public docs quality, evaluator trust, governed adoption
**Related Tickets:** T-048, T-049, T-050, T-051, T-052, T-053
**Goals:** G-DOCS-IA-001, G-001
**BDD Feature:** F-015-documentation-site-information-architecture.md
**Related Feature Contracts:** F-001, F-004, F-007, F-008, F-009, F-015
**Hypothesis:** A curated reader-first docs layer, backed by MarsDocSync and canonical harness docs, will make MARS easier to evaluate and safer to operate without weakening the repo's system-of-record model.
**Success Evidence:** Homepage first viewport explains MARS as a local AI product engineering team that can be inspected, governed, and improved; public docs route by safe action, proof need, governance evidence, operating recovery, and source-of-truth inspection; public pages identify safe actions, file-writing actions, ownership boundaries, evidence paths, and canonical source-of-truth docs; long catalog content is moved out of the homepage into a dedicated documentation map; DocSync, link sweep, docs consistency, and Go checks pass.
**Falsification Evidence:** Homepage still acts as a link wall; security, guardrails, and ownership evidence is not visible before command references; public docs and harness-consumed docs describe different truths; new docs duplicate canonical docs without source-of-truth labels.
**Scenario Schedule:** F-015-S001, F-015-S002, F-015-S003, F-015-S004, F-015-S005, F-015-S006, F-015-S007
**Current Failing Scenario:** None; F-015-S001 through F-015-S007 passed on 2026-06-29.
**Walking Skeleton Slice:** Create the docs product brief and update active goal, plan, feature, and ticket state before rewriting pages.
**Learning Or MVP Outcome:** Establish the documentation system contract before changing the public site.
**Created:** 2026-06-29
**Owner:** foundation-maintainer as Orchestrator with COO, Product/Docs Maintainer, Security/Governance Reviewer, QA, and Release Manager role packets
**Source:** Operator request to rebuild docs IA using the MARS planning operating model.

## Primary Outcome

Reframe the public documentation site as a trust-building product
documentation layer for anyone evaluating, operating, governing, or
maintaining MARS. The public site must explain what MARS is, who owns what,
what stays local, what agents can change, how guardrails work, how evidence is
reviewed, and where canonical source-of-truth docs live.

## Primary Pass Gate

The pass gate is green only when planning state, public pages, canonical links,
and checks agree:

- `docs/goals/active.md` includes G-DOCS-IA-001.
- This active exec plan schedules F-015 and T-048 through T-053.
- `docs/features/F-015-documentation-site-information-architecture.md`
  defines the reader-first behavior.
- Tickets T-048 through T-053 exist and map to the scenario schedule.
- `docs/product-specs/documentation-site.md` defines evaluation routes,
  reader actions, content model, DocSync relationship, risks, and before/after
  file changes.
- Public pages make security, ownership, guardrails, evidence, safe actions,
  and canonical docs visible before command catalogs.
- Verification covers link integrity, nav consistency, DocSync, docs
  consistency tests, and full Go tests.

## Agent Orchestrator Pattern

Codex main acts as `foundation-maintainer` and Orchestrator/integrator.
Role-assuming agents or role-labelled work packets own bounded lenses:

- COO/Product IA: goal, product brief, reader action, and proof-route
  coherence.
- Product/Docs Maintainer: public site IA, page templates, stable URLs,
  crosslinks, tone, and mobile readability.
- Security/Governance Reviewer: data locality, credential boundaries,
  guardrails, trust levels, ownership, evidence, recovery, and
  security-sensitive claims.
- QA: HTML links, anchors, primary nav consistency, DocSync, docs consistency,
  and Go tests.
- Release Manager: semantic commit, release notes, tag, release asset
  publication, verification, and push evidence.

Subagents audit or validate bounded slices. The Orchestrator integrates edits,
resolves conflicts, and owns final completion evidence.

## Scenario Schedule

| Scenario | Ticket | Outcome | Status |
| --- | --- | --- | --- |
| F-015-S001 | T-048 | Docs product brief defines evaluation routes, goals, reader actions, risks, and content model. | Passed |
| F-015-S002 | T-049 | Homepage is rebuilt as a trust-building front door. | Passed |
| F-015-S003 | T-050 | Documentation map separates public guides from canonical harness/system-record docs. | Passed |
| F-015-S004 | T-051 | Security, ownership, guardrails, and evidence guide serves governance review. | Passed |
| F-015-S005 | T-052 | Adoption guide supports safe trial, control review, pilot, proof, and rollout decisions. | Passed |
| F-015-S006 | T-053 | Existing guides are crosslinked to canonical docs and labelled by reader action. | Passed |
| F-015-S007 | T-053 | IA passes mobile, link, DocSync, and docs-consistency gates. | Passed |

## Before Filesystem Shape

| Current File/Area | Current Purpose |
| --- | --- |
| `docs/index.html` | Homepage, command chooser, guide list, full catalog, and source-record map in one page. |
| `docs/quickstart.html` | First run guide. |
| `docs/workflows.html` | Task-oriented user workflows. |
| `docs/*-guide.html`, `docs/*-reference.html` | Deep public guides and references. |
| `docs/harness-ecosystem/` | Separate adoption explainer with a different page model. |
| `docs/design-docs/` | Canonical architecture and operating decisions consumed by maintainers/agents. |
| `docs/features/` | Canonical BDD feature contracts. |
| `docs/goals/`, `docs/exec-plans/`, `docs/tickets/` | MARS planning and delivery system records. |
| `docs/validation/`, `docs/generated/`, `docs/runbooks/`, `docs/references/` | Evidence, generated-reference catalog, operational procedures, and research. |

## After Filesystem Shape

| File | Action | Purpose / Headline |
| --- | --- | --- |
| `docs/goals/active.md` | Update | Active goal: make the docs site explain, prove, and route MARS correctly. |
| `docs/exec-plans/active/current-operating-plan.md` | Update | Current plan: documentation site IA rebuild. |
| `docs/features/F-015-documentation-site-information-architecture.md` | New | The docs site helps readers trust, try, govern, and maintain MARS. |
| `docs/tickets/done/T-048-documentation-site-product-brief.md` | New, delivered | Define what the docs site is for. |
| `docs/tickets/done/T-049-rebuild-docs-homepage-ia.md` | New, delivered | Make the homepage a trust-building front door. |
| `docs/tickets/done/T-050-create-documentation-map.md` | New, delivered | Help readers find canonical MARS docs. |
| `docs/tickets/done/T-051-create-security-governance-guide.md` | New, delivered | Explain security, guardrails, ownership, and evidence. |
| `docs/tickets/done/T-052-create-adoption-guide.md` | New, delivered | Help readers evaluate MARS through evidence routes. |
| `docs/tickets/done/T-053-crosslink-existing-guides-to-canonical-docs.md` | New, delivered | Connect public guides to source-of-truth docs. |
| `docs/product-specs/documentation-site.md` | New | What the MARS docs site is for. |
| `docs/product-specs/product-surface.md` | Update | Current product surface includes the new docs IA. |
| `docs/index.html` | Rewrite | MARS is a local AI product engineering team you can inspect, govern, and improve. |
| `docs/documentation-map.html` | New | Find the canonical MARS document. |
| `docs/security-governance-guide.html` | New | Security, ownership, guardrails, and evidence. |
| `docs/adoption-guide.html` | New | Adopt MARS without losing control. |
| `docs/documentation-sync-guide.html` | Update | How human docs and harness docs stay in sync. |
| `docs/quickstart.html` | Light update | Run one safe MARS lifecycle. |
| `docs/workflows.html` | Light update | Common jobs MARS helps you complete. |
| `docs/planning-delivery-guide.html` | Light update | From idea to shipped change. |
| `docs/safety-quality-guide.html` | Light update | Trust, guardrails, quality, and recovery. |
| `docs/files-state-reference.html` | Light update | What MARS writes, owns, stores, and removes. |
| `docs/site.css` | Update | Layout support for evidence cards, trust pillars, doc-type badges, and before/after tables. |
| `docs/site.js` | Optional update | Stable page classification for the new IA pages. |
| `README.md` | Update | Point readers to the new homepage paths and documentation map. |

## Validation Gates

- `git diff --check`
- `node --check docs/site.js`
- recursive HTML link/anchor sweep for `docs/**/*.html`
- primary nav consistency assertion
- `mars docsync audit --repo .`
- `go test ./internal/docsconsistency ./internal/docsync`
- `go test ./...`
- release notes, tag, local release asset publication, and local release asset verification

## Current Evidence

- Remote trunk was fetched on 2026-06-29 and the temporary clean worktree was
  confirmed up to date with `origin/main` before edits.
- The preceding MARS rename active plan completed before this plan was
  scheduled. Its semantic commit, release-note commit, tag, and release-asset
  evidence remain in git history and release notes.
- Product/IA, Docs Maintainer, and Security/Governance role packets completed
  read-only briefs for planning conventions, public site IA, and governance
  claim boundaries.
- On 2026-06-29, `docs/product-specs/documentation-site.md` was updated to define
  the docs site purpose, evaluation routes, reader actions, content model,
  claim boundaries, DocSync relationship, and before/after filesystem shape.
- On 2026-06-29, `docs/index.html` was updated as a concise trust-building
  front door.
- `docs/documentation-map.html`, `docs/security-governance-guide.html`, and
  `docs/adoption-guide.html` were added on 2026-06-29 to provide the catalog,
  governance route, and evidence-based adoption route.
- Existing public guides were crosslinked on 2026-06-29 to the documentation
  map, governance guide, adoption guide, and canonical docs where they summarize
  source-of-truth rules.
- Operator IA feedback on 2026-06-29 removed explicit audience-lane routing.
  The homepage, adoption guide, documentation map, product spec, F-015, active
  goal, and ticket records were updated on 2026-06-29 to route by proof need,
  safe action, governance evidence, recovery, and source-of-truth inspection
  instead of organization type.
- Writing-style reference added:
  `docs/references/mdn-technical-writing.md`.
- PASS: `git diff --check`.
- PASS: `node --check docs/site.js`.
- PASS: recursive HTML link/anchor sweep checked 31 HTML files.
- PASS: primary nav consistency assertion checked 30 template HTML files.
- PASS: `GOCACHE=/private/tmp/mars-go-cache go run ./cmd/mars docsync audit --repo .`.
- PASS: `GOCACHE=/private/tmp/mars-go-cache go test ./internal/docsconsistency ./internal/docsync`.
- PASS: `GOCACHE=/private/tmp/mars-go-cache go test ./...`.

## Residual Risks

- This plan changes public documentation only. It does not implement generated
  docs generator behavior or any CLI/runtime behavior.
- The public site summarizes canonical docs. Where a public page and a
  canonical harness doc disagree, the canonical doc wins and this plan must fix
  the public summary.
