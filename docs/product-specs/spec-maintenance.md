# Product Spec Maintenance

**Status:** Accepted
**Updated:** 2026-05-02
**Owner:** Mars Harness maintainers
**Sources:** [product spec governance](../design-docs/product-spec-governance.md), [docs consistency tests](../../internal/docsconsistency)

## Why This Exists

Product specs had drifted into historical pitch material. Mars Harness needs them to be operational memory: concise enough for agents to read, comprehensive enough to guide product decisions, and checked enough that they evolve with the code.

## Definition Of Stale

A product spec is stale when it:

- names a command, generated file, role behavior, trust rule, scoring rule, setup behavior, or safety promise that no longer matches the code
- omits a public product surface that users or generated target repos rely on
- contradicts an accepted design doc
- describes an active hardening task as already complete
- leaves a new product spec out of [index.md](index.md)
- lacks status, update date, or owner metadata

## Required Update Flow

When a change affects product behavior:

1. Update the relevant product spec in the same semantic commit.
2. Update [index.md](index.md) if a spec is added, renamed, or retired.
3. Link to the design doc or exec plan that owns the implementation detail.
4. State why the behavior exists or changed, either directly or through the linked design doc.
5. Keep the product spec focused on promises and user-visible behavior.
6. Run the docs-consistency tests before committing.

## Mechanical Checks

`internal/docsconsistency` checks product specs for:

- strict trunk workflow language through the shared docs workflow test
- required metadata in every product spec
- catalog coverage for every product spec except the index itself
- valid markdown links from the product spec index

These checks do not prove the specs are perfect. They keep the directory navigable and prevent silent drift.

## Writing Rules

- Prefer present-tense product contracts over historical narrative.
- Mark partial capabilities honestly as "still hardening" or "planned".
- Link to design docs for rationale instead of duplicating implementation detail.
- Link to exec plans for remaining work instead of pretending it is complete.
- Keep source and generated-target behavior separate when they differ.
- Treat product specs as agent-readable routing documents, not marketing copy.

## Ownership

The maintainer or agent making a product-facing change owns the spec update. If the product implication is unclear, create or update an exec-plan task rather than leaving the mismatch undocumented.
