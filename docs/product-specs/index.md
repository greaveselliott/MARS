# Product Specs

**Status:** Accepted
**Updated:** 2026-05-20
**Owner:** Mars Harness maintainers

## Purpose

This directory is the living product contract for Mars Harness. It answers what the product promises, who it serves, what surfaces users and agents can rely on, and how the specs stay current as the harness evolves.

Design docs remain the place for architecture decisions. Exec plans remain the place for active delivery work. Product specs define the durable product shape those docs and plans are trying to serve.

## Spec Catalog

| Spec | Purpose | Update when |
| --- | --- | --- |
| [vision.md](vision.md) | Product promise, audience, principles, success measures, and north star. | The product promise, target user, tenets, or success measures change. |
| [product-surface.md](product-surface.md) | Current user-facing commands, generated artifacts, roles, safety model, scoring, trust, inference, and open hardening areas. | CLI behavior, generated target harness files, role semantics, scoring, trust, setup, doctor, safety, or integrations change. |
| [dashboard-control-plane.md](dashboard-control-plane.md) | Planned TanStack dashboard control plane contract: external Node prerequisite, local-admin auth, nonblocking APIs, Overview, Active Work, Preview, Agent Roster, Models, and GitHub-derived DORA. | Dashboard architecture, runtime prerequisites, auth, dashboard APIs, preview behavior, roster/model mutation proposals, or delivery metrics change. |
| [mars-relationship.md](mars-relationship.md) | How Mars informs Mars Harness and what supersession means. | Mars parity work changes, a Mars lesson is imported, or supersession criteria move. |
| [spec-maintenance.md](spec-maintenance.md) | How product specs self-document as the harness evolves. | Documentation governance, consistency checks, or product-spec ownership changes. |

## Freshness Contract

Every product spec must include:

- `**Status:**`
- `**Updated:** YYYY-MM-DD`
- `**Owner:**`

Every product spec except this index must be linked in the catalog above. The `internal/docsconsistency` package checks that requirement so new specs cannot be added silently.

Product-facing changes must update this directory when they affect:

- first-run setup or zero-config behavior
- generated target harness files
- CLI commands or public flags
- role, ticket, queue, trust, score, or telemetry semantics
- guardrail, safety, or tool policy behavior
- local inference defaults and doctor expectations
- optional remote-code-host integration behavior
- Mars parity and supersession commitments

## Reading Order

Start with [vision.md](vision.md), then [product-surface.md](product-surface.md). Use [dashboard-control-plane.md](dashboard-control-plane.md) for next-generation dashboard work, [mars-relationship.md](mars-relationship.md) when evaluating parity with Mars, and [spec-maintenance.md](spec-maintenance.md) when changing the product contract.
