# Documentation Site Product Spec

**Status:** Accepted
**Updated:** 2026-06-29
**Owner:** MARS maintainers
**Sources:** [product surface](product-surface.md), [vision](vision.md), [documentation sync architecture](../design-docs/documentation-sync-architecture.md), [F-015](../features/F-015-documentation-site-information-architecture.md), [MDN technical writing reference](../references/mdn-technical-writing.md)

## Purpose

The MARS documentation site is the public, human-facing front door for the
software factory. It should help a reader understand what MARS is, decide
whether it fits their risk model, try it safely, operate it, govern it, and
inspect the canonical records used by agents and maintainers.

The site does not replace harness-consumed docs. It explains and routes to them.

## Evaluation Routes

The site should work for solo users, teams, and security-sensitive
organizations without forcing them into named organization lanes. It routes by
the thing the reader needs to prove.

| Route | What The Reader Needs | First Useful Action |
| --- | --- | --- |
| Try safely | A local trial without surprise writes. | Run `mars doctor` and a dry-run role preview. |
| Evaluate fit | A clear view of product-team scope, ownership, and limits. | Read the homepage, adoption guide, and quickstart before mutating a repo. |
| Govern autonomy | Data boundaries, credential handling, guardrails, trust, and recovery. | Read security/governance, files/state, guardrails, and credentials docs. |
| Operate and recover | Day-to-day commands, dashboard, checks, troubleshooting, and rollback. | Use quickstart, workflows, operations, observability, and troubleshooting. |
| Extend and integrate | Model routes, tools, MCP, configuration, and validation boundaries. | Read models, tools/MCP, configuration, integrations, and validation docs. |
| Inspect canonical records | Planning, BDD, DocSync, release discipline, evidence, and source of truth. | Use the documentation map to open goals, active plan, features, tickets, and design docs. |

## Reader Actions

Every public page should make one or more actions clear:

- **Understand:** explain the concept plainly.
- **Evaluate:** show proof paths, risks, and limits.
- **Try safely:** distinguish read-only actions from writes.
- **Operate:** show normal commands and success signals.
- **Govern:** show ownership, guardrails, trust, evidence, and recovery.
- **Extend:** route to tools, MCP, configuration, roles, and integration docs.
- **Troubleshoot:** show symptoms, checks, and rollback or eject paths.
- **Inspect canonical records:** link source-of-truth docs with labels.

Public prose follows the writing standard captured in
[mdn-technical-writing.md](../references/mdn-technical-writing.md): clear
reader need, concise sections, consistent terms, logical progression, realistic
examples, descriptive links, and self-review before release.

## Content Model

| Content Type | Role | Canonical Status | Used By Agents |
| --- | --- | --- | --- |
| Public guide | Human-facing explanation and task flow. | Summary unless labelled otherwise. | Sometimes. |
| Public reference | Command, file, API, or configuration reference. | Summary of CLI/runtime behavior. | Sometimes. |
| Product spec | Durable product promise and surface. | Source of truth for product shape. | Yes. |
| Design doc | Architecture, decision, or operating doctrine. | Source of truth for why and how. | Yes. |
| BDD contract | Feature completeness and business logic. | Source of truth for done behavior. | Yes. |
| Goal, exec plan, ticket | Planning and delivery state. | Source of truth for active work. | Yes. |
| Validation evidence | Report, blocker, command, or replay path. | Source of truth for evidence. | Yes. |
| Generated reference | Reproducible source map or generated catalog. | Reference unless generator owns it. | Sometimes. |
| Runbook | Operational procedure. | Source of truth for that procedure. | Yes when routed. |

## Information Architecture

The homepage should be a trust-building front door, not a full catalog. It
should answer:

- What is MARS?
- What risk or proof question should the reader answer first?
- What stays local?
- What can agents change?
- How do guardrails and trust levels work?
- Where is the evidence?
- What can I do first without writing files?
- Where do I inspect canonical docs?

Long lists move to `docs/documentation-map.html`. Security, ownership,
guardrails, and evidence move to `docs/security-governance-guide.html`.
Safe trial, governance review, pilot, proof, and rollout decisions move to
`docs/adoption-guide.html`.

## Safety And Claim Boundaries

Use precise claims:

- MARS is local-first by default.
- Optional telemetry, GitHub, cloud model routes, and integrations are explicit
  configured surfaces.
- Guardrails include repo-owned YAML plus built-in policy. Hard YAML rules are
  syntactic checks; semantic security judgment still needs review.
- MARS provides auditability through commits, tickets, traces, scores,
  validation reports, release notes, and DocSync. It does not make LLM behavior
  deterministic.
- Scores and trust levels are operating signals, not compliance ratings.

Avoid absolute claims such as "no data ever leaves the machine", "secrets can
never leak", or "irreversible damage is impossible".

## MarsDocSync Relationship

Public docs pages that summarize operating doctrine should list this product
spec and the relevant canonical docs in their `MarsDocSync` block.

Code and style files that constrain public documentation behavior should list:

- `docs/product-specs/documentation-site.md`
- `docs/features/F-015-documentation-site-information-architecture.md`
- `docs/design-docs/documentation-sync-architecture.md`
- public pages affected by the behavior

`mars docsync audit --repo .` checks metadata shape and doc path existence. It
does not prove prose is semantically complete; reviewers still check that
summaries match canonical docs.

## Before And After Filesystem Shape

| Area | Before | After |
| --- | --- | --- |
| Homepage | Hero, command chooser, guide list, full catalog, and reference map on one page. | Trust-building front door with concise routing and safe first actions. |
| Catalog | Mixed into the homepage. | Dedicated documentation map grouped by purpose and canonical status. |
| Security and governance | Split across safety, auth, files/state, guardrails, and observability pages. | New guide routes control questions to canonical pages without hiding limits. |
| Adoption | Separate explainer plus scattered page links. | New adoption guide gives evidence routes, safe trial steps, pilot guidance, and proof paths. |
| Planning state | User chat and existing active plan. | Active goal, exec plan, F-015, tickets T-048 through T-053, and this product spec. |

## Success Measures

- A first-time reader can explain MARS as a local AI product engineering team.
- A security-sensitive reader can find local data boundaries, credential
  handling, guardrails, evidence, DocSync, and recovery before reading CLI
  reference.
- A beginner can identify read-only commands before file-writing commands.
- A maintainer or agent can find canonical planning, BDD, design, validation,
  and generated docs from the documentation map.
- The homepage is no longer a link wall.
- HTML links, primary nav, DocSync, docs consistency, and Go tests pass.
