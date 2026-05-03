# Generated Docs

**Status:** Accepted
**Updated:** 2026-05-02
**Owner:** Mars Harness maintainers

## Purpose

This directory is reserved for committed, reproducible reference documents generated from Mars Harness source state. It exists so agents can read stable summaries without re-scanning the whole codebase every run.

It is not a place for hand-written design decisions, product specs, tickets, or plans. Those belong in their normal directories.

## Current Status

No generated reference artifacts are committed yet. Model evaluation reports may
be generated under `docs/generated/model-evaluations/` by
`mars-harness models evaluate`; commit only reports that support a default-model
decision or benchmark record.

## Intended Artifacts

Future generated docs should be small, deterministic, and useful as context routes:

| Artifact | Source of truth | Why it belongs here |
| --- | --- | --- |
| Role registry | `.harness/manifest.yaml`, role prompts, trust defaults | Lets agents see available roles, tools, triggers, and chains quickly. |
| Tool inventory | `internal/tools/` registry | Lets agents inspect tool names, mutability, and policy constraints without reading every tool file. |
| Package map | `internal/` and `cmd/` packages | Gives agents a compact codebase map before deep file reads. |
| Model inventory | hardware registry and setup config | Shows pinned model choices, revisions, checksums, and hardware profile mapping. |
| Model evaluation reports | `mars-harness models evaluate` benchmark output | Records provider, model, hardware profile, benchmark pass/fail, timing, token counts, and promotion blockers for candidate review. |
| Score export | scoring database or exported score state | Makes role health visible in the repo when exported intentionally. |
| Bundle schema reference | bundle structs and examples | Documents generated target harness inputs without duplicating implementation docs. |

## Freshness Rules

Generated artifacts must:

- name the generator command or package that produced them
- name their source inputs
- include a generated timestamp or source revision when useful
- be deterministic enough for meaningful diffs
- be small enough to help context routing
- be cataloged in this README

If a generated artifact cannot be reproduced, it should not be committed here.

## Staleness Checks

`internal/docsconsistency` checks that:

- this README has status, update date, and owner metadata
- every markdown file under `docs/generated/` is cataloged here
- catalog links point to existing generated docs

Future generator commands should add stricter checks that regenerate artifacts in a temp directory and compare them with committed files.
