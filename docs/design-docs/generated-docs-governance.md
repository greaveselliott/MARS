# Generated Docs Governance

**Status:** Accepted
**Date:** 2026-05-02

## Context

`docs/generated/` existed as a placeholder, but no generator wrote to it and the README did not explain its product role. That made the directory look stale and ambiguous.

Mars Harness still needs a place for generated reference snapshots. OpenAI's harness guidance and the Mars parity plan both point toward compact, agent-readable maps: role registry, tool inventory, package map, model inventory, score export, and bundle schema reference. Those maps should help context routing without becoming hand-written docs that silently drift.

## Decisions

### AD-043: Generated Docs Are Reproducible Reference Snapshots

`docs/generated/` is reserved for committed outputs generated from source, manifests, databases, or explicit export commands. The directory should not contain hand-written decisions, tickets, product specs, or plans.

Each generated artifact must name its generator and source inputs. If it cannot be reproduced, it does not belong in `docs/generated/`.

### AD-044: Empty Generated Docs Are Acceptable Until A Generator Exists

The directory may contain only `README.md` while no generated artifacts exist. The README must say that plainly so agents do not infer that missing files are accidental.

### AD-045: Generated Docs Must Be Cataloged And Checked

The generated docs README is the catalog. Docs consistency tests verify README metadata, generated markdown catalog coverage, and catalog link validity. Future generator commands should add stronger freshness checks by regenerating artifacts and comparing them with committed outputs.

## Implementation Requirements

- Keep `docs/generated/README.md` as the catalog and freshness contract.
- Catalog every generated markdown artifact in `docs/generated/README.md`.
- Add docs-consistency coverage so new generated docs cannot be added silently.
- Add generator-specific tests when generator commands are introduced.

## Consequences

- `docs/generated/` is no longer an unexplained placeholder.
- Agents can distinguish generated reference maps from hand-written product and architecture docs.
- Future generated artifacts have a clear freshness bar before they are committed.
