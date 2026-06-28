# Product Spec Governance

**Status:** Accepted
**Date:** 2026-05-02

## Context

`docs/product-specs/` had fallen behind the product. It described the original pitch and Mars relationship, but not the current strict trunk workflow, generated target harness, zero-config inference expectations, trust and scoring commands, self-reflective telemetry, or spec maintenance rules.

MARS treats the repo as the system of record. Product promises cannot live only in chat or old plans. Agents need a current product map just as much as they need design docs and tickets.

## Decisions

### AD-040: Product Specs Are A Living Product Contract

Product specs define user-visible promises and product boundaries. They must be updated when the product surface changes, not only when a release is prepared.

Every product spec must include status, update date, and owner metadata. Every spec except the index must be cataloged from `docs/product-specs/index.md`.

### AD-041: Product Specs Link Out Instead Of Stuffing Context

Product specs should summarize product contracts and link to design docs, exec plans, references, and tickets for supporting detail. This keeps them useful as context routes and avoids turning the directory into a second architecture manual.

### AD-042: Product Spec Freshness Is Mechanically Checked

`internal/docsconsistency` verifies product-spec metadata, index coverage, index links, and strict trunk language. The check is intentionally small, but it turns the "keep specs current" requirement into a failing test when the directory structure drifts.

## Implementation Requirements

- `docs/product-specs/index.md` is the catalog and reading map.
- `docs/product-specs/product-surface.md` describes current commands, generated files, roles, scoring, trust, safety, local inference, optional integrations, and hardening areas.
- `docs/product-specs/spec-maintenance.md` defines stale specs and required update flow.
- Docs consistency tests fail when product specs are missing metadata or index coverage.

## Consequences

- Product-facing changes now carry a documentation obligation.
- Future agents can answer "what does MARS promise?" from a single indexed directory.
- Product specs stay smaller by routing to source docs instead of duplicating all rationale.
- The product contract can evolve without becoming invisible to the next run.
